package handler

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type ExchangeHandler struct{ db *gorm.DB }

func NewExchangeHandler(db *gorm.DB) *ExchangeHandler { return &ExchangeHandler{db: db} }

func (h *ExchangeHandler) nextCode(prefix, seqKey string) string {
	var setting model.SystemSetting
	if err := h.db.Where("key = ?", seqKey).First(&setting).Error; err != nil {
		setting = model.SystemSetting{Key: seqKey, Value: "0"}
	}
	seq, _ := strconv.Atoi(setting.Value)
	seq++
	setting.Value = strconv.Itoa(seq)
	h.db.Save(&setting)
	return fmt.Sprintf("%s%04d", prefix, seq)
}

// uniqueCode 序列号自愈：跳过已被占用的编号，避免唯一约束冲突导致静默失败
func (h *ExchangeHandler) uniqueCode(prefix, seqKey, table, column string) string {
	code := h.nextCode(prefix, seqKey)
	for {
		var cnt int64
		h.db.Table(table).Where(column+" = ?", code).Count(&cnt)
		if cnt == 0 {
			break
		}
		code = h.nextCode(prefix, seqKey)
	}
	return code
}

func (h *ExchangeHandler) ListRequests(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	q := h.db.Model(&model.ExchangeRequest{})
	if v := c.Query("call_sign"); v != "" { q = q.Where("call_sign LIKE ?", "%"+v+"%") }
	if v := c.Query("review_status"); v != "" { q = q.Where("review_status = ?", v) }
	if v := c.Query("scene_type"); v != "" { q = q.Where("scene_type = ?", v) }
	var total int64
	q.Count(&total)
	var items []model.ExchangeRequest
	q.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&items)
	response.Page(c, items, total, page, size)
}

func (h *ExchangeHandler) CreateRequest(c *gin.Context) {
	var req model.ExchangeRequest
	c.ShouldBindJSON(&req)
	req.CallSign = strings.ToUpper(req.CallSign)
	var count int64
	h.db.Model(&model.ExchangeRequest{}).Where("call_sign = ? AND review_status = 'PENDING'", req.CallSign).Count(&count)
	if count > 0 {
		response.Fail(c, 409, "该呼号已有待审核申请")
		return
	}
	req.RequestCode = randomCode(h.db, "exchange_requests", "request_code", "EX")
	if req.SceneType == "" {
		req.SceneType = "QSO"
	}
	req.SceneType = strings.ToUpper(req.SceneType)
	h.db.Create(&req)
	response.OK(c, req)
}

// defaultAddressText 自动回寄地址文本（只取一条）：优先台站档案地址（本台正式
// 回寄地址，可在"卡片版本/台站资料"维护），未配置时回退地址簿第一条；
// 两者皆无则不自动发送（保留管理员"重发地址"手动发送）
func (h *ExchangeHandler) defaultAddressText() (string, bool) {
	var p model.StationProfile
	if err := h.db.First(&p).Error; err == nil && strings.TrimSpace(p.Address) != "" {
		name := p.Name
		if strings.TrimSpace(name) == "" {
			name = p.CallSign
		}
		text := fmt.Sprintf("%s 收\n%s %s\n%s", name, p.PostalCode, "China", strings.TrimSpace(p.Address))
		return strings.TrimSpace(text), true
	}
	var first model.AddressBookEntry
	if err := h.db.Order("id ASC").First(&first).Error; err == nil && strings.TrimSpace(first.Address) != "" {
		name := first.Name
		if strings.TrimSpace(name) == "" {
			name = first.CallSign
		}
		text := fmt.Sprintf("%s 收\n%s %s\n%s", name, first.PostalCode, first.DestinationCountry, strings.TrimSpace(first.Address))
		return strings.TrimSpace(text), true
	}
	return "", false
}

// sendReturnAddressMail 发送回寄地址邮件（附进度/登记链接），返回错误文本（空串为成功）
func (h *ExchangeHandler) sendReturnAddressMail(c *gin.Context, ex *model.ExchangeRequest) string {
	if ex.Email == "" {
		return ""
	}
	link := siteURL(h.db, c) + "/status/" + ex.RequestCode
	subject := "QSL 换卡：回寄地址与寄出登记"
	body := fmt.Sprintf(
		"<p>您好 <b>%s</b>（%s）：</p>" +
		"<p>请将您的卡片寄至以下地址：</p>" +
		"<div style=\"background:#f8f7f2;border:1px dashed #d8d2c6;border-radius:6px;padding:14px 18px;margin:14px 0;white-space:pre-wrap;font-size:14px;line-height:1.8;\">%s</div>" +
		"<p>寄出后请点击下方按钮登记您的邮寄方式（挂号信填单号 / 平信免填），我们会即时收到记录：</p>" +
		"<p style=\"text-align:center;margin:24px 0;\"><a href=\"%s\" style=\"display:inline-block;background:#f5a623;color:#1a2d3d;padding:11px 30px;border-radius:4px;text-decoration:none;font-weight:bold;\">登记寄出信息</a></p>" +
		"<p style=\"color:#999;font-size:12px;word-break:break-all;\">如按钮无法点击，请复制链接：%s</p>" +
		"<p style=\"margin-bottom:0;\">73！</p>",
		htmlEscape(ex.Name), htmlEscape(ex.CallSign), htmlEscape(ex.ReturnAddressText), link, link)
	if err := sendMail(h.db, ex.Email, subject, body); err != nil {
		log.Printf("发送回寄地址邮件失败(%s): %v", ex.Email, err)
		return err.Error()
	}
	return ""
}

func (h *ExchangeHandler) Approve(c *gin.Context) {
	var item model.ExchangeRequest
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "申请不存在"); return
	}
	if item.ReviewStatus != "PENDING" {
		response.Fail(c, 422, "该申请已处理"); return
	}
	item.ReviewStatus = "APPROVED"
	item.ReviewedBy = c.GetString("username")
	item.ReviewedAt = cstNow().Format("2006-01-02 15:04:05")

	link := siteURL(h.db, c) + "/status/" + item.RequestCode

	// SWL 场景：对方先寄卡给我。审批通过即自动发送回寄地址（默认单条：台站
	// 档案优先，未配置时地址簿第一条）并合并进审批邮件，对方一封信即可完成寄件与登记。
	swl := strings.ToUpper(item.SceneType) == "SWL"
	addrAuto := ""
	if swl && item.ReturnAddressText == "" {
		if text, ok := h.defaultAddressText(); ok {
			addrAuto = text
			item.ReturnAddressText = text
			item.AddressSentAt = cstNow().Format("2006-01-02 15:04:05")
		}
	}
	h.db.Save(&item)

	if swl && item.Email != "" {
		subject := "您的换卡申请已通过"
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("<p>您好 <b>%s</b>（%s）：</p>", htmlEscape(item.Name), htmlEscape(item.CallSign)))
		sb.WriteString("<p>您的换卡申请已 <b style=\"color:#1f8f5f;\">审核通过</b>。</p>")
		if addrAuto != "" {
			sb.WriteString("<p>请将您的收听确认卡(QSL)寄至以下地址：</p>")
			sb.WriteString("<div style=\"background:#f8f7f2;border:1px dashed #d8d2c6;border-radius:6px;padding:14px 18px;margin:14px 0;white-space:pre-wrap;font-size:14px;line-height:1.8;\">" + htmlEscape(addrAuto) + "</div>")
			sb.WriteString("<p>寄出后请点击下方按钮登记您的邮寄方式（挂号信填单号 / 平信免填），我们会即时收到记录并尽快制卡回寄：</p>")
		} else {
			sb.WriteString("<p>接下来请按以下步骤操作：</p>")
			sb.WriteString("<ol style=\"padding-left:20px;margin:12px 0;\">")
			sb.WriteString("<li>等待我们向您发送<b>回寄地址</b>（将发送到本邮箱）</li>")
			sb.WriteString("<li>将您的收听确认卡(QSL)寄出</li>")
			sb.WriteString("<li>寄出后在进度页登记邮寄方式（挂号信/平信）与单号</li>")
			sb.WriteString("</ol>")
		}
		sb.WriteString("<p style=\"text-align:center;margin:26px 0;\"><a href=\"" + link + "\" style=\"display:inline-block;background:#f5a623;color:#1a2d3d;padding:11px 30px;border-radius:4px;text-decoration:none;font-weight:bold;\">查看申请进度 / 登记寄出</a></p>")
		sb.WriteString("<p style=\"color:#999;font-size:12px;word-break:break-all;\">如按钮无法点击，请复制链接：" + link + "</p>")
		sb.WriteString("<p style=\"margin-bottom:0;\">73！</p>")
		if err := sendMail(h.db, item.Email, subject, sb.String()); err != nil {
			log.Printf("SWL 审批邮件发送失败(%s): %v", item.Email, err)
		}
	}
	response.OK(c, item)
}

func (h *ExchangeHandler) Reject(c *gin.Context) {
	var item model.ExchangeRequest
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "申请不存在"); return
	}
	var req struct{ ReviewReason string `json:"review_reason"` }
	c.ShouldBindJSON(&req)
	if item.ReviewStatus == "REJECTED" { response.Fail(c, 422, "该申请已是拒绝状态"); return }
	item.ReviewStatus = "REJECTED"
	item.ReviewReason = req.ReviewReason
	item.ReviewedAt = cstNow().Format("2006-01-02 15:04:05")
	h.db.Save(&item)
	// 拒绝后邮件通知对方（附原因与重新申请引导）
	if item.Email != "" {
		reason := strings.TrimSpace(item.ReviewReason)
		if reason == "" {
			reason = "（未填写具体原因）"
		}
		link := siteURL(h.db, c) + "/apply"
		subject := "您的换卡申请未通过"
		body := fmt.Sprintf(
			"<p>您好 <b>%s</b>（%s）：</p>" +
			"<p>很遗憾，您的换卡申请<b style=\"color:#c0392b;\">未通过审核</b>。</p>" +
			"<div style=\"background:#fdf3f2;border-left:3px solid #c0392b;padding:12px 16px;margin:14px 0;\"><b>拒绝原因：</b>%s</div>" +
			"<p>欢迎核对信息后重新提交申请。</p>" +
			"<p style=\"text-align:center;margin:24px 0;\"><a href=\"%s\" style=\"display:inline-block;background:#f5a623;color:#1a2d3d;padding:11px 30px;border-radius:4px;text-decoration:none;font-weight:bold;\">重新申请</a></p>" +
			"<p style=\"margin-bottom:0;\">73！</p>",
			htmlEscape(item.Name), htmlEscape(item.CallSign), htmlEscape(reason), link)
		if err := sendMail(h.db, item.Email, subject, body); err != nil {
			log.Printf("拒绝通知邮件发送失败(%s): %v", item.Email, err)
		}
	}
	response.OK(c, item)
}

// buildCardForExchange 由换卡申请构建回寄卡片记录（不落库；QSO 场景顺带关联通联记录）
func (h *ExchangeHandler) buildCardForExchange(ex *model.ExchangeRequest) model.CardRecord {
	now := cstNow()
	sceneType := strings.ToUpper(ex.SceneType)
	if sceneType == "" {
		sceneType = "QSO"
	}

	// 构建业务备注：换卡理由 + 场景证据摘要
	businessRemarks := fmt.Sprintf("换卡理由: %s\n场景: %s", ex.ApplicationReason, sceneType)
	switch sceneType {
	case "QSO":
		businessRemarks += fmt.Sprintf("\n通联日期: %s %s\n频率: %s %s\n模式: %s",
			ex.QsoDate, ex.QsoTime, ex.QsoFreq, ex.QsoBand, ex.QsoMode)
	case "EYEBALL":
		businessRemarks += fmt.Sprintf("\n见面日期: %s %s\n活动: %s\n地点: %s",
			ex.EyeballDate, ex.EyeballTime, ex.EyeballActivity, ex.EyeballLocation)
	case "SWL":
		businessRemarks += fmt.Sprintf("\n收听日期: %s %s\n频率: %s %s\n模式: %s",
			ex.SwlDate, ex.SwlTime, ex.SwlFreq, ex.SwlBand, ex.SwlMode)
	}
	if ex.UseBureau {
		businessRemarks += fmt.Sprintf("\n对方卡片局: %s", ex.BureauName)
	}

	card := model.CardRecord{
		CardCode:        randomCode(h.db, "card_records", "card_code", "C"),
		CallSign:        ex.CallSign,
		OwnerName:       ex.Name,
		CardType:        sceneType,
		SceneType:       sceneType,
		CardVersion:     ex.CardVersion,
		CardDate:        now.Format("2006-01-02"),
		CardTime:        now.Format("15:04"),
		CardRemarks:     "期待与您空中相遇。\nLooking forward to meeting you on the air.",
		BusinessRemarks: businessRemarks,
		MailType:        "REGISTERED",
		MailTargetEmail: ex.Email,
		FlowStatus:      "PENDING_ISSUE",
	}

	// QSO 场景：尝试关联已有通联记录
	if sceneType == "QSO" && ex.QsoDate != "" {
		var qso model.QsoRecord
		if err := h.db.Where("call_sign = ? AND date = ?", ex.CallSign, ex.QsoDate).First(&qso).Error; err == nil {
			card.QsoRecordID = &qso.ID
			qso.HasCard = true
			h.db.Save(&qso)
		}
	}
	return card
}

func (h *ExchangeHandler) CreateCard(c *gin.Context) {
	var ex model.ExchangeRequest
	if err := h.db.First(&ex, c.Param("id")).Error; err != nil {
		response.NotFound(c, "申请不存在"); return
	}
	if ex.ReviewStatus != "APPROVED" { response.Fail(c, 422, "只有已通过的申请才能创建卡片"); return }
	if ex.CardCreated { response.Fail(c, 422, "该申请已创建卡片"); return }

	card := h.buildCardForExchange(&ex)
	if err := h.db.Create(&card).Error; err != nil {
		response.Fail(c, 500, "创建卡片失败，请稍后重试")
		return
	}
	ex.CardCreated = true
	ex.CardCreatedAt = cstNow().Format("2006-01-02 15:04:05")
	ex.CreatedCardID = &card.ID
	h.db.Save(&ex)

	// SWL 手动建卡兜底（新流程由确认收卡自动建卡）：建卡后邮件通知对方（附状态查询链接）
	if strings.ToUpper(ex.SceneType) == "SWL" && ex.Email != "" {
		link := siteURL(h.db, c) + "/status/" + ex.RequestCode
		subject := "您的回寄卡片已建单"
		body := fmt.Sprintf(
			"<p>您好 <b>%s</b>（%s）：</p>" +
			"<p>您的回寄卡片已完成建单（卡片编号 <b>%s</b>），后续将打包寄出到您预留的地址。</p>" +
			"<p>寄出后您可在下方链接查看申请进度与物流信息：</p>" +
			"<p style=\"text-align:center;margin:24px 0;\"><a href=\"%s\" style=\"display:inline-block;background:#f5a623;color:#1a2d3d;padding:11px 30px;border-radius:4px;text-decoration:none;font-weight:bold;\">查看申请进度</a></p>" +
			"<p style=\"color:#999;font-size:12px;word-break:break-all;\">如按钮无法点击，请复制链接：%s</p>" +
			"<p style=\"margin-bottom:0;\">73！</p>",
			htmlEscape(ex.Name), htmlEscape(ex.CallSign), card.CardCode, link, link)
		if err := sendMail(h.db, ex.Email, subject, body); err != nil {
			log.Printf("SWL 回寄卡建卡通知邮件发送失败(%s): %v", ex.Email, err)
		}
	}
	response.OK(c, card)
}

func (h *ExchangeHandler) ListActivities(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	var total int64
	h.db.Model(&model.OfflineActivity{}).Count(&total)
	var items []model.OfflineActivity
	h.db.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&items)
	response.Page(c, items, total, page, size)
}

func (h *ExchangeHandler) CreateActivity(c *gin.Context) {
	var req model.OfflineActivity
	if err := c.ShouldBindJSON(&req); err != nil { response.Fail(c, 400, "参数错误"); return }
	req.ActivityCode = h.uniqueCode("ACT-", "offline_activity_sequence", "offline_activities", "activity_code")
	h.db.Create(&req)
	response.OK(c, req)
}

func (h *ExchangeHandler) UpdateActivity(c *gin.Context) {
	var item model.OfflineActivity
	if err := h.db.First(&item, c.Param("id")).Error; err != nil { response.NotFound(c, "活动不存在"); return }
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil { response.Fail(c, 400, "参数错误"); return }
	h.db.Model(&item).Updates(req)
	response.OK(c, item)
}

// ReceiveReturnCard SWL 反寄收卡登记：我方收到对方寄来的卡，
// 生成收卡记录（收到的卡归属收卡记录），并自动创建回寄卡片记录（进入卡片记录
// 列表待发卡，无需再手动"创建卡片"），随后邮件通知对方
func (h *ExchangeHandler) ReceiveReturnCard(c *gin.Context) {
	var ex model.ExchangeRequest
	if err := h.db.First(&ex, c.Param("id")).Error; err != nil {
		response.NotFound(c, "申请不存在")
		return
	}
	if ex.ReturnMailType == "" {
		response.Fail(c, 422, "对方尚未登记寄出信息")
		return
	}
	if ex.ReturnReceivedAt != "" {
		response.Fail(c, 422, "该申请已登记过收卡")
		return
	}
	var req struct {
		Remarks string `json:"remarks"`
	}
	c.ShouldBindJSON(&req)

	// 收卡记录编号：与收卡记录页同序列（加载行更新，避免静默失败）
	var setting model.SystemSetting
	if err := h.db.Where("key = ?", "receive_record_sequence").First(&setting).Error; err != nil {
		setting = model.SystemSetting{Key: "receive_record_sequence", Value: "0"}
	}
	seq, _ := strconv.Atoi(setting.Value)
	seq++
	setting.Value = strconv.Itoa(seq)
	h.db.Save(&setting)
	now := cstNow()
	code := fmt.Sprintf("R%04d-%s", seq, now.Format("20060102"))

	record := model.ReceiveRecord{
		ReceiveCode:  code,
		CallSign:     ex.CallSign,
		CardType:     "SWL",
		BusinessType: "SWL",
		ReceivedDate: now.Format("2006-01-02"),
		MatchStatus:  "UNMATCHED",
		Remarks:      strings.TrimSpace("SWL 反寄收卡 " + ex.RequestCode + "；" + req.Remarks),
	}
	if err := h.db.Create(&record).Error; err != nil {
		response.Error(c, "收卡记录创建失败，请稍后重试")
		return
	}

	ex.ReturnReceivedAt = now.Format("2006-01-02 15:04:05")

	// 确认收件后自动创建回寄卡片记录：卡片直接进入卡片记录列表（待发卡），
	// 管理员在卡片记录里写卡/发卡即可，收件人与换卡申请邮箱一致
	card := h.buildCardForExchange(&ex)
	cardErr := h.db.Create(&card).Error
	if cardErr == nil {
		ex.CardCreated = true
		ex.CardCreatedAt = now.Format("2006-01-02 15:04:05")
		ex.CreatedCardID = &card.ID
	} else {
		log.Printf("SWL 收卡自动建卡失败(%s): %v", ex.RequestCode, cardErr)
	}
	h.db.Save(&ex)

	mailErr := ""
	if ex.Email != "" {
		link := siteURL(h.db, c) + "/status/" + ex.RequestCode
		subject := "您的卡片已收到，回寄卡片已建单"
		tracking := ""
		if ex.ReturnMailType == "REGISTERED" {
			tracking = "（您的单号：" + htmlEscape(ex.ReturnTracking) + "）"
		}
		cardLine := ""
		if cardErr == nil {
			cardLine = "，回寄卡片已完成建单（卡片编号 <b>" + card.CardCode + "</b>）"
		}
		body := fmt.Sprintf(
			"<p>您好 <b>%s</b>（%s）：</p>" +
			"<p>我们已<b style=\"color:#1f8f5f;\">收到您寄出的卡片</b>%s，收卡记录编号 <b>%s</b>%s。</p>" +
			"<p>回寄卡片制卡寄出后，您可在下方链接查看申请进度与物流信息。</p>" +
			"<p style=\"text-align:center;margin:24px 0;\"><a href=\"%s\" style=\"display:inline-block;background:#f5a623;color:#1a2d3d;padding:11px 30px;border-radius:4px;text-decoration:none;font-weight:bold;\">查看申请进度</a></p>" +
			"<p style=\"color:#999;font-size:12px;word-break:break-all;\">如按钮无法点击，请复制链接：%s</p>" +
			"<p style=\"margin-bottom:0;\">73！</p>",
			htmlEscape(ex.Name), htmlEscape(ex.CallSign), tracking, code, cardLine, link, link)
		if err := sendMail(h.db, ex.Email, subject, body); err != nil {
			mailErr = err.Error()
			log.Printf("SWL 收卡通知邮件发送失败(%s): %v", ex.Email, err)
		}
	}
	resp := gin.H{"receive_code": code, "return_received_at": ex.ReturnReceivedAt, "mail_error": mailErr}
	if cardErr == nil {
		resp["card_code"] = card.CardCode
	}
	response.OK(c, resp)
}

// SendAddress 从"我的地址"中多选若干条，发送到申请人邮箱，并把地址文本留在申请上
// （公开进度页会展示，供对方寄卡使用；SMTP 未配置时邮箱发送失败但留档仍生效）
func (h *ExchangeHandler) SendAddress(c *gin.Context) {
	var ex model.ExchangeRequest
	if err := h.db.First(&ex, c.Param("id")).Error; err != nil {
		response.NotFound(c, "申请不存在")
		return
	}
	var req struct {
		AddressIDs []uint `json:"address_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.AddressIDs) == 0 {
		response.Fail(c, 400, "请选择要发送的地址")
		return
	}
	var entries []model.AddressBookEntry
	if err := h.db.Where("id IN ?", req.AddressIDs).Find(&entries).Error; err != nil || len(entries) == 0 {
		response.Fail(c, 404, "地址不存在")
		return
	}
	var sb strings.Builder
	for i, e := range entries {
		sb.WriteString(fmt.Sprintf("地址 %d：\n%s 收\n%s %s\n%s\n\n", i+1, e.Name, e.PostalCode, e.DestinationCountry, e.Address))
	}
	ex.ReturnAddressText = strings.TrimSpace(sb.String())
	ex.AddressSentAt = cstNow().Format("2006-01-02 15:04:05")
	h.db.Save(&ex)

	// 邮件复用统一发送逻辑（审批通过时已自动发送过一次，此处用于重发/改发）
	mailErr := h.sendReturnAddressMail(c, &ex)
	response.OK(c, gin.H{"address_text": ex.ReturnAddressText, "address_sent_at": ex.AddressSentAt, "mail_error": mailErr})
}
