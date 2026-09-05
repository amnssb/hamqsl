package handler

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

func refreshStatus(c *model.CardRecord) {
	if c.CardType != "" && len(c.CardType) > 6 && c.CardType[len(c.CardType)-6:] == "(ERROR)" {
		c.FlowStatus = "ERROR"
	} else if c.ReceiptConfirmed { c.FlowStatus = "SIGNED"
	} else if c.CardReceived { c.FlowStatus = "RECEIVED"
	} else if c.CardSent { c.FlowStatus = "SENT"
	} else if c.EnvelopePrinted { c.FlowStatus = "PACKED"
	} else if c.CardIssued { c.FlowStatus = "ISSUED"
	} else { c.FlowStatus = "PENDING_ISSUE" }
}

func (h *CardHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }

	q := h.db.Model(&model.CardRecord{})
	if v := c.Query("call_sign"); v != "" { q = q.Where("call_sign LIKE ?", "%"+v+"%") }
	if v := c.Query("scene_type"); v != "" { q = q.Where("scene_type = ?", v) }
	if v := c.Query("flow_status"); v != "" { q = q.Where("flow_status = ?", v) }
	if v := c.Query("mail_type"); v != "" { q = q.Where("mail_type = ?", v) }
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
	cardCode := randomCode(h.db, "card_records", "card_code", "C")
	card := model.CardRecord{
		CardCode: cardCode, CallSign: req.CallSign,
		OwnerName: req.OwnerName,
		CardType: req.CardType, SceneType: req.SceneType, CardVersion: req.CardVersion,
		QsoRecordID: req.QsoRecordID, MailType: req.MailType, MailTargetEmail: req.MailTargetEmail,
		BusinessRemarks: req.BusinessRemarks, CardRemarks: req.CardRemarks,
		CardDate: now.Format("2006-01-02"), CardTime: now.Format("15:04"),
	}
	refreshStatus(&card)
	h.db.Create(&card)

	if req.QsoRecordID != nil {
		h.db.Model(&model.QsoRecord{}).Where("id = ?", *req.QsoRecordID).Update("has_card", true)
	}

	// 卡片记录页手动建卡：自动关联同呼号、同场景、已通过且未建卡的换卡申请，
	// 回写 card_created/created_card_id。否则该申请在线上换卡仍显示「创建卡片」
	// 按钮，再次点击会重复建卡并发送重复的建卡邮件（卡"回到"线上换卡的现象）
	if req.CallSign != "" && req.SceneType != "" {
		var ex model.ExchangeRequest
		if err := h.db.Where("call_sign = ? AND scene_type = ? AND review_status = ? AND card_created = ?",
			req.CallSign, req.SceneType, "APPROVED", false).Order("id DESC").First(&ex).Error; err == nil {
			ex.CardCreated = true
			ex.CardCreatedAt = now.Format("2006-01-02 15:04:05")
			ex.CreatedCardID = &card.ID
			h.db.Save(&ex)
		}
	}
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
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	h.db.Model(&item).Updates(req)
	refreshStatus(&item)
	h.db.Save(&item)
	response.OK(c, item)
}

func (h *CardHandler) Delete(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	// 事务：清理指向该卡片的收卡记录（回寄确认自动生成的那条），再删卡片本身
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("outbound_card_id = ?", item.ID).Delete(&model.ReceiveRecord{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.CardRecord{}, item.ID).Error
	})
	if err != nil {
		response.Error(c, "删除失败")
		return
	}
	response.OKMsg(c, "删除成功")
}

func (h *CardHandler) Issue(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	item.CardIssued = true
	item.CardIssuedAt = cstNow().Format("2006-01-02 15:04:05")
	refreshStatus(&item)
	h.db.Save(&item)
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
	item.CardSent = true
	item.SentAt = cstNow().Format("2006-01-02 15:04:05")
	item.SentRemarks = req.SentRemarks
	if req.MailType != "" { item.MailType = req.MailType }
	if req.TrackingNumber != "" { item.TrackingNumber = req.TrackingNumber }
	if req.TrackingCarrier != "" { item.TrackingCarrier = req.TrackingCarrier }
	refreshStatus(&item)

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
	item.CardReceived = true
	item.ReceivedAt = req.ReceivedDate
	item.ReceivedRemarks = req.ReceivedRemarks
	refreshStatus(&item)
	h.db.Save(&item)
	response.OK(c, item)
}

func (h *CardHandler) Resend(c *gin.Context) {
	var item model.CardRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	item.CardIssued = false
	item.CardIssuedAt = ""
	item.EnvelopePrinted = false
	item.CardSent = false
	item.SentAt = ""
	item.TrackingNumber = ""
	item.TrackingStatus = ""
	item.TrackingDetail = ""
	item.TrackingUpdatedAt = ""
	item.CreatedMailStatus = ""
	item.SentMailStatus = ""
	refreshStatus(&item)
	h.db.Save(&item)
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
		if err := json.Unmarshal([]byte(item.TrackingDetail), &details); err != nil { details = []gin.H{{"time": item.TrackingUpdatedAt, "status": item.TrackingDetail}} }
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
	return "<p style=\"text-align:center;margin:24px 0;\"><a href=\"" + link + "\" style=\"display:inline-block;background:#f5a623;color:#1a2d3d;padding:11px 30px;border-radius:4px;text-decoration:none;font-weight:bold;\">查看申请进度</a></p>" +
		"<p style=\"color:#999;font-size:12px;word-break:break-all;\">如按钮无法点击，请复制链接：" + link + "</p>"
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
	if err := h.db.First(&item, c.Param("id")).Error; err != nil { response.NotFound(c, "卡片不存在"); return }
	var req struct { Scene string `json:"scene"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.Fail(c, 400, "参数错误"); return }
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
	if qso.HasCard {
		response.Fail(c, 422, "该通联记录已建卡")
		return
	}
	if req.MailType != "ORDINARY" {
		req.MailType = "REGISTERED"
	}
	now := time.Now()
	card := model.CardRecord{
		CardCode:        randomCode(h.db, "card_records", "card_code", "C"),
		CallSign:        qso.CallSign,
		OwnerName:       qso.Operator,
		CardType:        "QSO",
		SceneType:       "QSO",
		CardVersion:     req.CardVersion,
		QsoRecordID:     &qso.ID,
		CardDate:        now.Format("2006-01-02"),
		CardTime:        now.Format("15:04"),
		CardRemarks:     req.CardRemarks,
		MailType:        req.MailType,
		MailTargetEmail: req.MailTargetEmail,
		FlowStatus:      "PENDING_ISSUE",
	}
	if err := h.db.Create(&card).Error; err != nil {
		response.Error(c, "创建卡片失败，请稍后重试")
		return
	}
	qso.HasCard = true
	h.db.Save(&qso)
	response.OK(c, card)
}

func (h *CardHandler) BatchTrackingUpdate(c *gin.Context) {
	var cards []model.CardRecord
	h.db.Where("card_sent = ? AND mail_type = ? AND tracking_number <> ''", true, "REGISTERED").Find(&cards)
	for i := range cards {
		if cards[i].TrackingStatus == "" { cards[i].TrackingStatus = "查询中"; cards[i].TrackingUpdatedAt = cstNow().Format("2006-01-02 15:04:05"); h.db.Save(&cards[i]) }
	}
	response.OK(c, gin.H{"updated": len(cards)})
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
	item.TrackingUpdatedAt = time.Now().UTC().Format("2006-01-02 15:04:05")
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
	item.ReturnMailEnabled = *req.Enabled
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
	if card.ReturnMailedAt == "" {
		response.Fail(c, 422, "对方尚未登记回寄，无法确认收卡")
		return
	}
	if card.ReturnReceivedAt != "" {
		response.Fail(c, 422, "该卡回寄已确认过收卡（记录 "+card.ReturnRecordCode+"）")
		return
	}

	receivedDate := cstNow().Format("2006-01-02")
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
		ReceivedAt:     cstNow().Format("2006-01-02 15:04:05"),
		OutboundCardID: &outboundID,
		MatchStatus:    "MATCHED",
		MatchReason:    "回寄收卡：对方确认收件并回寄（卡片 " + card.CardCode + "）",
		Remarks:        "对方回寄登记" + returnKind + " " + card.ReturnMailedAt,
	}
	if err := h.db.Create(&record).Error; err != nil {
		response.Fail(c, 500, "写入收卡记录失败："+err.Error())
		return
	}
	card.ReturnReceivedAt = cstNow().Format("2006-01-02 15:04:05")
	card.ReturnRecordCode = code
	if err := h.db.Save(&card).Error; err != nil {
		response.Fail(c, 500, "更新卡片回寄状态失败："+err.Error())
		return
	}
	response.OK(c, gin.H{"receive_code": code, "return_received_at": card.ReturnReceivedAt})
}
