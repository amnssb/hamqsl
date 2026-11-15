package handler

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/flow"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type CardHandler struct{ db *gorm.DB }

func NewCardHandler(db *gorm.DB) *CardHandler { return &CardHandler{db: db} }

// randomCode 生成随机编号：prefix + 8 位无歧义大写字母数字（去除 0/O/1/I/L），查库保证唯一
func randomCode(db *gorm.DB, table, column, prefix string) string {
	const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
	for {
		b := make([]byte, 8)
		if _, err := crand.Read(b); err != nil {
			return fmt.Sprintf("%s%d", prefix, time.Now().UnixNano())
		}
		for i := range b {
			b[i] = alphabet[int(b[i])%len(alphabet)]
		}
		code := prefix + string(b)
		var cnt int64
		db.Table(table).Where(column+" = ?", code).Count(&cnt)
		if cnt == 0 {
			return code
		}
	}
}

func (h *CardHandler) List(c *gin.Context) {
	page, size := pageParams(c)

	q := h.db.Model(&model.CardRecord{})
	if v := c.Query("call_sign"); v != "" {
		q = q.Where("call_sign LIKE ?", callSignPattern(v))
	}
	if v := c.Query("scene_type"); v != "" {
		q = q.Where("scene_type = ?", v)
	}
	if v := c.Query("flow_status"); v != "" {
		q = q.Where("flow_status = ?", v)
	}
	if v := c.Query("mail_type"); v != "" {
		q = q.Where("mail_type = ?", v)
	}
	// 总览「已发卡片」是累计寄出数（含之后签收的），不能用 flow_status=SENT，
	// 那个状态只剩「已寄出但还没签收」。
	if v := c.Query("card_sent"); v == "true" || v == "false" {
		q = q.Where("card_sent = ?", v == "true")
	}
	// 回寄状态筛选：enabled=已开启待对方寄回 / mailed=对方已寄出待我签收 / received=回寄已收
	switch c.Query("return_status") {
	case "enabled":
		q = q.Where("return_mail_enabled = ? AND (return_mailed_at IS NULL OR return_mailed_at = '')", true)
	case "mailed":
		q = q.Where("return_mailed_at != ? AND (return_received_at IS NULL OR return_received_at = '')", "")
	case "received":
		q = q.Where("return_received_at != ?", "")
	}

	var total int64
	q.Count(&total)
	var items []model.CardRecord
	q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items)
	response.Page(c, items, total, page, size)
}

func (h *CardHandler) Create(c *gin.Context) {
	var req struct {
		CallSign        string `json:"call_sign"`
		OwnerName       string `json:"owner_name"`
		CardType        string `json:"card_type"`
		SceneType       string `json:"scene_type"`
		CardVersion     string `json:"card_version"`
		QsoRecordID     *uint  `json:"qso_record_id"`
		MailType        string `json:"mail_type"`
		MailTargetEmail string `json:"mail_target_email"`
		BusinessRemarks string `json:"business_remarks"`
		CardRemarks     string `json:"card_remarks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	now := cstNow()
	card := model.CardRecord{
		CardCode: randomCode(h.db, "card_records", "card_code", "C"), CallSign: req.CallSign,
		OwnerName: req.OwnerName,
		CardType:  req.CardType, SceneType: req.SceneType, CardVersion: req.CardVersion,
		QsoRecordID: req.QsoRecordID, MailType: req.MailType, MailTargetEmail: req.MailTargetEmail,
		BusinessRemarks: req.BusinessRemarks, CardRemarks: req.CardRemarks,
		CardDate: flow.Day(now), CardTime: now.Format("15:04"),
	}
	// 建卡、回写 has_card、绑上已通过的换卡申请，都在 flow 的同一事务里。
	if err := flow.CreateManual(h.db, &card, flow.Stamp(now)); err != nil {
		response.Error(c, "创建卡片失败，请稍后重试")
		return
	}
	invalidatePublicStatsCache()
	response.OK(c, card)
}

func (h *CardHandler) Get(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	response.OK(c, item)
}

func (h *CardHandler) Update(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	// 只允许改非流程字段（白名单见 patch.go）：卡片状态必须走 issue/sent/received 端点，
	// 否则一个 PUT 就能把 flow_status 直接改成 SIGNED。
	ok, err := applyPatch(c, h.db, &item, c.Param("id"), cardEditableFields)
	if !ok {
		if err != nil {
			response.Fail(c, 500, "保存失败："+err.Error())
			return
		}
		response.Fail(c, 400, "没有可更新的字段")
		return
	}
	// 白名单改不了进度。这里按 flow_status 把布尔位对齐，避免历史脏数据继续分歧。
	if flow.Align(&item) {
		if err := h.db.Save(&item).Error; err != nil {
			response.Fail(c, 500, "保存失败："+err.Error())
			return
		}
	}
	response.OK(c, item)
}

func (h *CardHandler) Delete(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	// 删卡、解开换卡申请、恢复通联 has_card 在同一事务里，由状态机完成。
	if err := flow.DeleteCard(h.db, item.ID); err != nil {
		response.Error(c, "删除失败")
		return
	}
	invalidatePublicStatsCache()
	response.OKMsg(c, "删除成功")
}

func (h *CardHandler) Issue(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	if err := flow.Issue(&item, flow.Stamp(cstNow())); err != nil {
		if writeFlowErr(c, err) {
			return
		}
		response.Error(c, "制卡失败")
		return
	}
	if err := h.db.Save(&item).Error; err != nil {
		response.Error(c, "制卡失败")
		return
	}
	response.OK(c, item)
}

func (h *CardHandler) ConfirmSent(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	var req struct {
		SentRemarks     string `json:"sent_remarks"`
		MailType        string `json:"mail_type"`
		TrackingNumber  string `json:"tracking_number"`
		TrackingCarrier string `json:"tracking_carrier"`
	}
	c.ShouldBindJSON(&req)
	if err := flow.Send(&item, flow.SendInput{
		At:              flow.Stamp(cstNow()),
		Remarks:         req.SentRemarks,
		MailType:        req.MailType,
		TrackingNumber:  req.TrackingNumber,
		TrackingCarrier: req.TrackingCarrier,
	}); err != nil {
		if writeFlowErr(c, err) {
			return
		}
		response.Error(c, "发信失败")
		return
	}

	// 发卡后自动邮件提醒收卡人（配置了邮箱或关联换卡申请时）
	email, ex := h.resolveCardNotify(&item)
	if email != "" {
		link := ""
		if ex != nil {
			link = siteURL(h.db, c) + "/status/" + ex.RequestCode
		}
		subject, body := cardMailContent(&item, ex, link, "sent")
		if err := sendMail(h.db, email, subject, body); err != nil {
			item.SentMailStatus = "FAILED"
			log.Printf("发卡通知邮件发送失败(%s): %v", email, err)
		} else {
			item.SentMailStatus = "SENT"
		}
	}

	h.db.Save(&item)
	invalidatePublicStatsCache()
	response.OK(c, item)
}

func (h *CardHandler) ConfirmReceived(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	var req struct {
		ReceivedDate    string `json:"received_date"`
		ReceivedRemarks string `json:"received_remarks"`
	}
	c.ShouldBindJSON(&req)
	if err := flow.Receive(&item, flow.ReceiveInput{
		At:            req.ReceivedDate,
		Remarks:       req.ReceivedRemarks,
		UpdateRemarks: true,
	}); err != nil {
		if writeFlowErr(c, err) {
			return
		}
		response.Error(c, "收卡登记失败")
		return
	}
	if err := h.db.Save(&item).Error; err != nil {
		response.Error(c, "收卡登记失败")
		return
	}
	invalidatePublicStatsCache()
	response.OK(c, item)
}

func (h *CardHandler) Resend(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	if err := flow.Resend(&item); err != nil {
		if writeFlowErr(c, err) {
			return
		}
		response.Error(c, "重发失败")
		return
	}
	if err := h.db.Save(&item).Error; err != nil {
		response.Error(c, "重发失败")
		return
	}
	response.OK(c, item)
}

func (h *CardHandler) GetTracking(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	details := []gin.H{}
	if item.TrackingDetail != "" {
		if err := json.Unmarshal([]byte(item.TrackingDetail), &details); err != nil {
			details = []gin.H{{"time": item.TrackingUpdatedAt, "status": item.TrackingDetail}}
		}
	}
	response.OK(c, gin.H{
		"tracking_number": item.TrackingNumber, "carrier": item.TrackingCarrier,
		"status": item.TrackingStatus, "details": details,
		"updated_at": item.TrackingUpdatedAt,
	})
}

// resolveCardNotify 解析卡片通知的收件人邮箱与关联换卡申请（用于状态链接）
func (h *CardHandler) resolveCardNotify(item *model.CardRecord) (string, *model.ExchangeRequest) {
	var ex model.ExchangeRequest
	hasEx := h.db.Where("created_card_id = ?", item.ID).First(&ex).Error == nil
	email := item.MailTargetEmail
	if email == "" && hasEx {
		email = ex.Email
	}
	if hasEx {
		return email, &ex
	}
	return email, nil
}

// linkBlock 状态链接按钮（无链接时返回空串）
func cardLinkBlock(link string) string {
	if link == "" {
		return ""
	}
	safe := htmlEscape(link)
	return "<p style=\"text-align:center;margin:24px 0;\"><a href=\"" + safe + "\" style=\"display:inline-block;background:#f5a623;color:#1a2d3d;padding:11px 30px;border-radius:4px;text-decoration:none;font-weight:bold;\">查看申请进度</a></p>" +
		"<p style=\"color:#999;font-size:12px;word-break:break-all;\">如按钮无法点击，请复制链接：" + safe + "</p>"
}

// cardMailContent 按场景生成卡片通知的标题与 HTML 正文
func cardMailContent(item *model.CardRecord, ex *model.ExchangeRequest, link, scene string) (string, string) {
	track := ""
	if item.MailType == "REGISTERED" && item.TrackingNumber != "" {
		track = "，挂号信号码 <b>" + htmlEscape(item.TrackingNumber) + "</b>"
	}
	// 称呼：姓名优先（申请/通联记录带过来的 owner_name），呼号随其后；
	// 无关联申请的通联建卡也至少带上对方呼号，不再出现空称呼"您好："
	hello := "<p>您好" + func() string {
		call := item.CallSign
		if ex != nil && ex.CallSign != "" {
			call = ex.CallSign
		}
		name := item.OwnerName
		if ex != nil && strings.TrimSpace(ex.Name) != "" {
			name = ex.Name
		}
		out := ""
		if strings.TrimSpace(name) != "" {
			out += " <b>" + htmlEscape(name) + "</b>"
		}
		if call != "" {
			out += "（<b>" + htmlEscape(call) + "</b>）"
		}
		return out
	}() + "：</p>"
	switch scene {
	case "created":
		return "您的卡片已建单", hello + "<p>为您准备的 QSL 卡片已建单（卡片编号 <b>" + item.CardCode + "</b>" + track + "），将尽快寄出。</p>" + cardLinkBlock(link) + "<p style=\"margin-bottom:0;\">73！</p>"
	case "received":
		return "卡片签收确认", hello + "<p>我们已确认您收到 QSL 卡片（编号 <b>" + item.CardCode + "</b>）。感谢本次换卡，期待与您空中再会！</p>" + cardLinkBlock(link) + "<p style=\"margin-bottom:0;\">73！</p>"
	case "tracking":
		status := item.TrackingStatus
		if status == "" {
			status = "查询中"
		}
		return "快递状态更新", hello + "<p>您的 QSL 卡片（编号 <b>" + item.CardCode + "</b>）快递状态：<b>" + htmlEscape(status) + "</b>" + track + "。</p>" + cardLinkBlock(link) + "<p style=\"margin-bottom:0;\">73！</p>"
	default: // sent
		return "您的卡片已寄出", hello + "<p>您的 QSL 卡片（编号 <b>" + item.CardCode + "</b>）已寄出" + track + "，请注意查收。</p>" + cardLinkBlock(link) + "<p style=\"margin-bottom:0;\">73！</p>"
	}
}

// SendMail 手动发送卡片状态通知邮件（真实发送）
func (h *CardHandler) SendMail(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	var req struct {
		Scene string `json:"scene"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	scene := req.Scene
	if scene == "" {
		scene = "sent"
	}
	email, ex := h.resolveCardNotify(&item)
	if email == "" {
		response.Fail(c, 422, "该卡片未配置收件邮箱，且未关联换卡申请")
		return
	}
	link := ""
	if ex != nil {
		link = siteURL(h.db, c) + "/status/" + ex.RequestCode
	}
	subject, body := cardMailContent(&item, ex, link, scene)
	if err := sendMail(h.db, email, subject, body); err != nil {
		h.db.Model(&item).Update("sent_mail_status", "FAILED")
		response.Fail(c, 500, "发送失败: "+err.Error())
		return
	}
	if scene == "created" {
		h.db.Model(&item).Update("created_mail_status", "SENT")
	} else {
		h.db.Model(&item).Update("sent_mail_status", "SENT")
	}
	response.OKMsg(c, "邮件已发送至 "+email)
}

// FromQso 从通联记录创建卡片（主动寄卡场景），并回写 has_card
func (h *CardHandler) FromQso(c *gin.Context) {
	var req struct {
		QsoRecordID     uint   `json:"qso_record_id"`
		CardVersion     string `json:"card_version"`
		MailType        string `json:"mail_type"`
		MailTargetEmail string `json:"mail_target_email"`
		CardRemarks     string `json:"card_remarks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.QsoRecordID == 0 {
		response.Fail(c, 400, "请选择通联记录")
		return
	}
	var qso model.QsoRecord
	if err := h.db.First(&qso, req.QsoRecordID).Error; err != nil {
		response.NotFound(c, "通联记录不存在")
		return
	}
	if req.MailType != "ORDINARY" {
		req.MailType = "REGISTERED"
	}
	now := cstNow()
	card := model.CardRecord{
		CardCode:        randomCode(h.db, "card_records", "card_code", "C"),
		OwnerName:       qso.Operator,
		CardVersion:     req.CardVersion,
		CardDate:        flow.Day(now),
		CardTime:        now.Format("15:04"),
		CardRemarks:     req.CardRemarks,
		MailType:        req.MailType,
		MailTargetEmail: req.MailTargetEmail,
	}
	// 场景校验、has_card 的条件更新和卡片插入在同一事务里。
	if err := flow.CreateFromQSO(h.db, &qso, &card); err != nil {
		if writeFlowErr(c, err) {
			return
		}
		response.Error(c, "创建卡片失败，请稍后重试")
		return
	}
	response.OK(c, card)
}

// BatchTrackingUpdate 把"已挂号寄出但还没有快递状态"的卡片批量标记为查询中。
//
// 原来是 Find 出全部行再逐条 Save（N 条 UPDATE，每条都回写整行）。
// 改单条 UPDATE：条件里带上 tracking_status = ”，语义等价但只有 1 次写。
func (h *CardHandler) BatchTrackingUpdate(c *gin.Context) {
	res := h.db.Model(&model.CardRecord{}).
		Where("card_sent = ? AND mail_type = ? AND tracking_number <> '' AND tracking_status = ''",
			true, "REGISTERED").
		Updates(map[string]interface{}{
			"tracking_status":     "查询中",
			"tracking_updated_at": cstNow().Format("2006-01-02 15:04:05"),
		})
	if res.Error != nil {
		response.Error(c, "批量更新失败："+res.Error.Error())
		return
	}
	response.OK(c, gin.H{"updated": res.RowsAffected})
}

func (h *CardHandler) UpdateTracking(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	var req struct {
		TrackingNumber  string `json:"tracking_number"`
		TrackingCarrier string `json:"tracking_carrier"`
	}
	c.ShouldBindJSON(&req)
	item.TrackingNumber = req.TrackingNumber
	item.TrackingCarrier = req.TrackingCarrier
	// 与 ensureTrackingFresh 写入同一口径（东八区墙钟）：
	// 这里原来是 time.Now().UTC()，比 cstNow() 早 8 小时，导致缓存判断自己跟自己打架。
	item.TrackingUpdatedAt = cstNow().Format(cstLayout)
	h.db.Save(&item)
	response.OK(c, item)
}

// ToggleReturnMail 后台按卡开启/关闭回寄功能：开启后对方确认收件页才显示回寄引导
func (h *CardHandler) ToggleReturnMail(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		response.Fail(c, 400, "enabled 必填（true/false）")
		return
	}
	flow.SetReturnEnabled(&item, *req.Enabled)
	if err := h.db.Save(&item).Error; err != nil {
		response.Fail(c, 500, "保存失败")
		return
	}
	response.OK(c, gin.H{"id": item.ID, "return_mail_enabled": item.ReturnMailEnabled})
}

// ReceiveReturnMail 后台确认收到对方回寄的卡：自动写入一条收卡记录（R 编号序列），
// 关联本次发出的卡片；卡片记录标记回寄签收时间与关联记录编号，防重复确认。
func (h *CardHandler) ReceiveReturnMail(c *gin.Context) {
	var card model.CardRecord
	if err := h.db.First(&card, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	if err := flow.PrepareCardReturn(&card); err != nil {
		if writeFlowErr(c, err) {
			return
		}
		response.Error(c, "确认回寄失败")
		return
	}

	now := cstNow()
	receivedDate := flow.Day(now)
	code := nextReceiveCode(h.db, receivedDate)
	businessType := card.SceneType
	if businessType == "" {
		businessType = "QSO"
	}
	outboundID := card.ID
	returnKind := "平信"
	if card.ReturnMailType == "REGISTERED" {
		returnKind = "挂号信 单号" + card.ReturnTracking
	}
	record := model.ReceiveRecord{
		ReceiveCode:    code,
		CallSign:       card.CallSign,
		CardType:       card.CardType,
		BusinessType:   businessType,
		ReceivedDate:   receivedDate,
		ReceivedAt:     flow.Stamp(now),
		OutboundCardID: &outboundID,
		MatchStatus:    "MATCHED",
		MatchReason:    "回寄收卡：对方确认收件并回寄（卡片 " + card.CardCode + "）",
		Remarks:        "对方回寄登记" + returnKind + " " + card.ReturnMailedAt,
	}
	if err := flow.CommitCardReturn(h.db, &card, &record, flow.Stamp(now)); err != nil {
		if writeFlowErr(c, err) {
			return
		}
		response.Fail(c, 500, "写入收卡记录失败："+err.Error())
		return
	}
	response.OK(c, gin.H{"receive_code": code, "return_received_at": card.ReturnReceivedAt})
}
