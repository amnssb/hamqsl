package handler

import (
	"fmt"
	"log"
	"strings"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type PublicHandler struct{ db *gorm.DB }

func NewPublicHandler(db *gorm.DB) *PublicHandler { return &PublicHandler{db: db} }

func (h *PublicHandler) GetCardByCode(c *gin.Context) {
	cardCode := c.Param("card_code")
	if cardCode == "" {
		response.Fail(c, 400, "卡片编号必填")
		return
	}
	var card model.CardRecord
	if err := h.db.Where("card_code = ?", cardCode).First(&card).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	response.OK(c, gin.H{
		"card_code":          card.CardCode,
		"call_sign":          card.CallSign,
		"card_type":          card.CardType,
		"card_version":       card.CardVersion,
		"flow_status":        card.FlowStatus,
		"mail_type":          card.MailType,
		"tracking_number":    card.TrackingNumber,
		"card_sent":          card.CardSent,
		"card_received":      card.CardReceived,
		"receipt_confirmed":  card.ReceiptConfirmed,
		"card_date":          card.CardDate,
		"return_mail_enabled": card.ReturnMailEnabled,
		"return_mail_type":   card.ReturnMailType,
		"return_tracking":    card.ReturnTracking,
		"return_mailed_at":   card.ReturnMailedAt,
	})
}

// ReturnMail 公开回寄登记：对方确认收卡后，在确认页获取本台地址并把他的卡寄回，
// 在此登记邮寄方式与单号。仅允许对"已确认收件"的卡片登记；成功后邮件提醒通知邮箱。
func (h *PublicHandler) ReturnMail(c *gin.Context) {
	var req struct {
		CardCode       string `json:"card_code"`
		MailType       string `json:"mail_type"`
		TrackingNumber string `json:"tracking_number"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数不正确")
		return
	}
	cardCode := strings.TrimSpace(req.CardCode)
	mailType := strings.ToUpper(strings.TrimSpace(req.MailType))
	tracking := strings.TrimSpace(req.TrackingNumber)
	if cardCode == "" || mailType == "" {
		response.Fail(c, 400, "卡片编号与邮寄方式必填")
		return
	}
	if mailType != "REGISTERED" && mailType != "ORDINARY" {
		response.Fail(c, 400, "邮寄方式不正确（REGISTERED / ORDINARY）")
		return
	}
	if mailType == "REGISTERED" && tracking == "" {
		response.Fail(c, 400, "挂号信请填写单号")
		return
	}
	var card model.CardRecord
	if err := h.db.Where("card_code = ?", cardCode).First(&card).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}
	if !card.ReceiptConfirmed {
		response.Fail(c, 422, "请先确认收件，再登记回寄")
		return
	}
	if !card.ReturnMailEnabled {
		response.Fail(c, 422, "本次寄件未开通回寄服务，请联系台站管理员")
		return
	}
	card.ReturnMailType = mailType
	card.ReturnTracking = tracking
	card.ReturnMailedAt = cstNow().Format("2006-01-02 15:04:05")
	if err := h.db.Save(&card).Error; err != nil {
		response.Fail(c, 500, "登记失败，请稍后重试")
		return
	}

	// 通知邮箱提醒（未配置则跳过；发送失败仅记日志不影响登记结果）
	if notify := getSettingValue(h.db, "notify_email"); notify != "" {
		typeName := "平信"
		if mailType == "REGISTERED" {
			typeName = "挂号信"
		}
		trackingLine := ""
		if tracking != "" {
			trackingLine = "<br>单号：<b>" + htmlEscape(tracking) + "</b>"
		}
		subject := "对方已回寄卡片：" + card.CallSign + "（" + card.CardCode + "）"
		body := mailWrap("对方已回寄卡片",
			"<p>卡片 <b>"+htmlEscape(card.CardCode)+"</b>（"+htmlEscape(card.CallSign)+"）的持有人已确认收件并登记回寄：</p>"+
				"<p>邮寄方式：<b>"+typeName+"</b>"+trackingLine+"<br>登记时间："+card.ReturnMailedAt+"</p>"+
				"<p>请留意查收回寄的卡片。</p>")
		if err := sendMail(h.db, notify, subject, body); err != nil {
			log.Printf("[mail] 回寄提醒发送失败 notify=%s err=%v", notify, err)
		}
	}

	response.OK(c, gin.H{
		"card_code":         card.CardCode,
		"return_mail_type":  card.ReturnMailType,
		"return_tracking":   card.ReturnTracking,
		"return_mailed_at":  card.ReturnMailedAt,
	})
}

// StationMailInfo 公开返回本台回寄地址（仅限寄件所需字段；空配置返回空对象不 404）。
// 地址来源：后台勾选的地址簿条目（设置 return_address_id）优先，其次台站信息。
func (h *PublicHandler) StationMailInfo(c *gin.Context) {
	if idStr := strings.TrimSpace(getSettingValue(h.db, "return_address_id")); idStr != "" {
		var ab model.AddressBookEntry
		if err := h.db.Where("id = ?", idStr).First(&ab).Error; err == nil {
			response.OK(c, gin.H{
				"call_sign":   ab.CallSign,
				"name":        ab.Name,
				"name_en":     "",
				"postal_code": ab.PostalCode,
				"address":     ab.Address,
				"address_en":  "",
			})
			return
		}
	}
	var p model.StationProfile
	if err := h.db.First(&p).Error; err != nil {
		response.OK(c, gin.H{})
		return
	}
	response.OK(c, gin.H{
		"call_sign":   p.CallSign,
		"name":        p.Name,
		"name_en":     p.NameEn,
		"postal_code": p.PostalCode,
		"address":     p.Address,
		"address_en":  p.AddressEn,
	})
}

// ExchangeStatus 公开查询申请实时进度（按申请编号；仅返回进度所需字段）
func (h *PublicHandler) ExchangeStatus(c *gin.Context) {
	code := c.Param("request_code")
	var ex model.ExchangeRequest
	if err := h.db.Where("request_code = ?", code).First(&ex).Error; err != nil {
		response.NotFound(c, "申请不存在")
		return
	}
	out := gin.H{
		"request_code":  ex.RequestCode,
		"call_sign":     ex.CallSign,
		"scene_type":    ex.SceneType,
		"review_status": ex.ReviewStatus,
		"review_reason": ex.ReviewReason,
		"reviewed_at":   ex.ReviewedAt,
		"card_created":  ex.CardCreated,
		"created_at":    ex.CreatedAt,

		// SWL 反寄流程
		"return_address_text": ex.ReturnAddressText,
		"address_sent_at":     ex.AddressSentAt,
		"return_mail_type":    ex.ReturnMailType,
		"return_tracking":     ex.ReturnTracking,
		"return_mailed_at":    ex.ReturnMailedAt,
		"return_received_at":  ex.ReturnReceivedAt,
	}
	if ex.CardCreated && ex.CreatedCardID != nil {
		var card model.CardRecord
		if err := h.db.First(&card, *ex.CreatedCardID).Error; err == nil {
			out["card_code"] = card.CardCode
			out["flow_status"] = card.FlowStatus
			out["card_sent"] = card.CardSent
			out["tracking_number"] = card.TrackingNumber
			out["tracking_carrier"] = card.TrackingCarrier
			out["tracking_status"] = card.TrackingStatus
			out["tracking_details"] = parseTrackingDetail(card.TrackingDetail)
		}
	}
	response.OK(c, out)
}

// PublicStats 公开统计（仅聚合数量，不含任何个人信息）：首页实时数据面板
func (h *PublicHandler) PublicStats(c *gin.Context) {
	var sent, signed, pending int64
	h.db.Model(&model.CardRecord{}).Where("card_sent = ?", true).Count(&sent)
	h.db.Model(&model.CardRecord{}).Where("receipt_confirmed = ?", true).Count(&signed)
	h.db.Model(&model.ExchangeRequest{}).Where("review_status = ?", "PENDING").Count(&pending)
	response.OK(c, gin.H{
		"cards_sent":       sent,
		"cards_signed":     signed,
		"pending_requests": pending,
	})
}

// SiteInfo 公开返回站点名称、发件邮箱（白名单引导用）与站点公告（未配置返回空）
func (h *PublicHandler) SiteInfo(c *gin.Context) {
	name := strings.TrimSpace(getSettingValue(h.db, "site_name"))
	if name == "" {
		name = "QSL 卡片管理系统"
	}
	// 发件邮箱跟随 SMTP 设置：发件人地址优先，留空回退用户名（与 sendMail 一致）
	sender := strings.TrimSpace(getSettingValue(h.db, "smtp_from"))
	if sender == "" {
		sender = strings.TrimSpace(getSettingValue(h.db, "smtp_user"))
	}
	notice := strings.TrimSpace(getSettingValue(h.db, "site_notice"))
	response.OK(c, gin.H{"site_name": name, "sender_email": sender, "site_notice": notice})
}

func (h *PublicHandler) StationCards(c *gin.Context) {
	var cards []model.StationCard
	h.db.Where("is_active = true").Order("sort_order").Find(&cards)
	response.OK(c, cards)
}

func (h *PublicHandler) Bureaus(c *gin.Context) {
	var items []model.BureauEntry
	h.db.Order("bureau_name").Find(&items)
	response.OK(c, items)
}

func (h *PublicHandler) SubmitExchange(c *gin.Context) {
	var req struct {
		CallSign          string `json:"call_sign"`
		SceneType         string `json:"scene_type"`
		CardVersion       string `json:"card_version"`
		UseBureau         bool   `json:"use_bureau"`
		BureauName        string `json:"bureau_name"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		Telephone         string `json:"telephone"`
		PostalCode        string `json:"postal_code"`
		Address           string `json:"address"`
		Remarks           string `json:"remarks"`
		ApplicationReason string `json:"application_reason"`
		// QSO 场景
		QsoDate    string `json:"qso_date"`
		QsoTime    string `json:"qso_time"`
		QsoFreq    string `json:"qso_freq"`
		QsoBand    string `json:"qso_band"`
		QsoMode    string `json:"qso_mode"`
		QsoRstSent string `json:"qso_rst_sent"`
		QsoRstRcvd string `json:"qso_rst_rcvd"`
		// EYEBALL 场景
		EyeballDate     string `json:"eyeball_date"`
		EyeballTime     string `json:"eyeball_time"`
		EyeballActivity string `json:"eyeball_activity"`
		EyeballLocation string `json:"eyeball_location"`
		EyeballType     string `json:"eyeball_type"`
		// SWL 场景
		SwlDate string `json:"swl_date"`
		SwlTime string `json:"swl_time"`
		SwlFreq string `json:"swl_freq"`
		SwlBand string `json:"swl_band"`
		SwlMode string `json:"swl_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	if req.CallSign == "" || req.Email == "" {
		response.Fail(c, 400, "呼号和邮箱必填")
		return
	}
	// SWL 场景地址用于接收我方回寄卡片，各场景均必填
	if req.Address == "" {
		response.Fail(c, 400, "呼号、邮箱和地址必填")
		return
	}
	if req.UseBureau && strings.TrimSpace(req.BureauName) == "" {
		response.Fail(c, 400, "通过卡片局收卡时，请填写您的卡片局名称")
		return
	}

	// 场景类型校验
	sceneType := strings.ToUpper(req.SceneType)
	switch sceneType {
	case "QSO":
		if req.QsoDate == "" || req.QsoFreq == "" || req.QsoMode == "" {
			response.Fail(c, 400, "QSO 场景需要通联日期、频率和模式")
			return
		}
	case "EYEBALL":
		if req.EyeballDate == "" || req.EyeballActivity == "" {
			response.Fail(c, 400, "见面场景需要见面日期和活动名称")
			return
		}
	case "SWL":
		if req.SwlDate == "" || req.SwlFreq == "" || req.SwlMode == "" {
			response.Fail(c, 400, "SWL 场景需要收听日期、频率和模式")
			return
		}
	default:
		response.Fail(c, 400, "请选择有效的场景类型")
		return
	}

	var count int64
	h.db.Model(&model.ExchangeRequest{}).Where("call_sign = ? AND review_status = 'PENDING'", strings.ToUpper(req.CallSign)).Count(&count)
	if count > 0 {
		response.Fail(c, 409, "该呼号已有待审核申请")
		return
	}

	requestCode := randomCode(h.db, "exchange_requests", "request_code", "EX")

	ex := model.ExchangeRequest{
		RequestCode:       requestCode,
		SceneType:         sceneType,
		CallSign:          strings.ToUpper(req.CallSign),
		CardVersion:       req.CardVersion,
		UseBureau:         req.UseBureau,
		BureauName:        req.BureauName,
		Email:             req.Email,
		Name:              req.Name,
		Telephone:         req.Telephone,
		PostalCode:        req.PostalCode,
		Address:           req.Address,
		Remarks:           req.Remarks,
		ApplicationReason: req.ApplicationReason,
		QsoDate:           req.QsoDate,
		QsoTime:           req.QsoTime,
		QsoFreq:           req.QsoFreq,
		QsoBand:           req.QsoBand,
		QsoMode:           req.QsoMode,
		QsoRstSent:        req.QsoRstSent,
		QsoRstRcvd:        req.QsoRstRcvd,
		EyeballDate:       req.EyeballDate,
		EyeballTime:       req.EyeballTime,
		EyeballActivity:   req.EyeballActivity,
		EyeballLocation:   req.EyeballLocation,
		EyeballType:       strings.ToUpper(req.EyeballType),
		SwlDate:           req.SwlDate,
		SwlTime:           req.SwlTime,
		SwlFreq:           req.SwlFreq,
		SwlBand:           req.SwlBand,
		SwlMode:           req.SwlMode,
		ReviewStatus:      "PENDING",
	}
	if err := h.db.Create(&ex).Error; err != nil {
		response.Fail(c, 500, "申请提交失败，请稍后重试")
		return
	}
	// 新申请邮件通知管理员（设置中配置了通知邮箱时）
	if notifyEmail := getSettingValue(h.db, "notify_email"); notifyEmail != "" {
		sceneName := map[string]string{"QSO": "QSO 通联", "SWL": "SWL 收听", "EYEBALL": "EYEBALL 见面"}[ex.SceneType]
		eyeballTag := ""
		if ex.SceneType == "EYEBALL" && ex.EyeballType == "ONLINE" {
			eyeballTag = "（网络EYE）"
		}
		subject := "新换卡申请：" + ex.CallSign + "（" + sceneName + "）"
		body := fmt.Sprintf(
			"<p>收到一条新的换卡申请：</p>" +
			"<div style=\"background:#f8f7f2;border-left:3px solid #f5a623;padding:12px 16px;margin:14px 0;line-height:2;\">" +
			"申请编号：<b>%s</b><br>呼号：<b>%s</b>%s<br>场景：%s<br>姓名：%s<br>邮箱：%s<br>" +
			"申请理由：%s</div>" +
			"<p>请登录管理后台及时处理。</p>",
			ex.RequestCode, htmlEscape(ex.CallSign), eyeballTag, htmlEscape(sceneName), htmlEscape(ex.Name), htmlEscape(ex.Email), htmlEscape(ex.ApplicationReason))
		if err := sendMail(h.db, notifyEmail, subject, body); err != nil {
			log.Printf("新申请通知邮件发送失败(%s): %v", notifyEmail, err)
		}
	}
	response.OK(c, ex)
}

func (h *PublicHandler) ConfirmReceipt(c *gin.Context) {
	var req struct {
		CardCode     string `json:"card_code"`
		CallSign     string `json:"call_sign"`
		ReceivedDate string `json:"received_date"`
		Remarks      string `json:"remarks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	if req.CardCode == "" || req.ReceivedDate == "" {
		response.Fail(c, 400, "卡片编号和收件日期必填")
		return
	}

	var card model.CardRecord
	if err := h.db.Where("card_code = ?", req.CardCode).First(&card).Error; err != nil {
		response.NotFound(c, "卡片不存在")
		return
	}

	if req.CallSign != "" && !strings.EqualFold(card.CallSign, req.CallSign) {
		response.Fail(c, 400, "呼号与卡片记录不匹配")
		return
	}

	card.CardReceived = true
	card.ReceiptConfirmed = true
	card.ReceivedAt = req.ReceivedDate
	card.ReceivedRemarks = req.Remarks
	if card.CardType != "" && len(card.CardType) > 6 && card.CardType[len(card.CardType)-6:] == "(ERROR)" {
		card.FlowStatus = "ERROR"
	} else if card.ReceiptConfirmed { card.FlowStatus = "SIGNED"
	} else { card.FlowStatus = "RECEIVED" }
	h.db.Save(&card)

	// "我发出的卡"由对方签收确认：仅推进卡片状态（SIGNED），不再生成收卡记录——
	// 收卡记录只登记对方主动寄来的卡（SWL 反寄收卡 / 手动登记）
	response.OK(c, gin.H{
		"card_code":     card.CardCode,
		"call_sign":     card.CallSign,
		"received_date": req.ReceivedDate,
		"flow_status":   card.FlowStatus,
	})
}

// SubmitReturnMail SWL 反寄：对方寄出后登记邮寄方式与单号，管理端即时可见
func (h *PublicHandler) SubmitReturnMail(c *gin.Context) {
	var req struct {
		RequestCode    string `json:"request_code"`
		MailType       string `json:"mail_type"`
		TrackingNumber string `json:"tracking_number"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	if req.RequestCode == "" {
		response.Fail(c, 400, "申请编号必填")
		return
	}
	mailType := strings.ToUpper(req.MailType)
	if mailType == "" {
		mailType = "ORDINARY"
	}
	if mailType != "REGISTERED" && mailType != "ORDINARY" {
		response.Fail(c, 400, "邮寄方式无效")
		return
	}
	tracking := strings.TrimSpace(req.TrackingNumber)
	if mailType == "REGISTERED" && tracking == "" {
		response.Fail(c, 400, "挂号信请填写单号")
		return
	}
	var ex model.ExchangeRequest
	if err := h.db.Where("request_code = ?", req.RequestCode).First(&ex).Error; err != nil {
		response.NotFound(c, "申请不存在")
		return
	}
	ex.ReturnMailType = mailType
	ex.ReturnTracking = tracking
	ex.ReturnMailedAt = cstNow().Format("2006-01-02 15:04:05")
	if err := h.db.Save(&ex).Error; err != nil {
		response.Fail(c, 500, "登记失败，请稍后重试")
		return
	}
	response.OK(c, gin.H{
		"request_code":    ex.RequestCode,
		"return_mail_type": ex.ReturnMailType,
		"return_tracking":  ex.ReturnTracking,
		"return_mailed_at": ex.ReturnMailedAt,
	})
}

func (h *PublicHandler) QueryTracking(c *gin.Context) {
	trackingNumber := c.Query("tracking_number")
	if trackingNumber == "" { trackingNumber = c.Param("tracking_number") }
	if trackingNumber == "" {
		response.Fail(c, 400, "请输入单号")
		return
	}

	var card model.CardRecord
	if err := h.db.Where("tracking_number = ?", trackingNumber).First(&card).Error; err != nil {
		response.NotFound(c, "未找到该单号对应的卡片")
		return
	}

	note := ensureTrackingFresh(h.db, &card)
	response.OK(c, gin.H{
		"tracking_number": card.TrackingNumber,
		"carrier": card.TrackingCarrier,
		"status": card.TrackingStatus,
		"details": parseTrackingDetail(card.TrackingDetail),
		"updated_at": card.TrackingUpdatedAt,
		"auto": kuaidi100Configured(h.db),
		"note": note,
	})
}
