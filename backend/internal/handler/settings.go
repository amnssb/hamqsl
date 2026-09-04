package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type SettingsHandler struct{ db *gorm.DB }

func NewSettingsHandler(db *gorm.DB) *SettingsHandler { return &SettingsHandler{db: db} }

func (h *SettingsHandler) getSetting(key string) string {
	var s model.SystemSetting
	if err := h.db.Where("key = ?", key).First(&s).Error; err != nil { return "" }
	return s.Value
}

// setSetting 按 key upsert 配置；Save 对 ID=0 的 struct 走 INSERT，
// 会撞 key 唯一索引并静默失败，导致已有配置永远无法更新
func (h *SettingsHandler) setSetting(key, val string) {
	var s model.SystemSetting
	if err := h.db.Where("key = ?", key).First(&s).Error; err != nil {
		h.db.Create(&model.SystemSetting{Key: key, Value: val})
		return
	}
	h.db.Model(&s).Update("value", val)
}

func (h *SettingsHandler) GetSmtp(c *gin.Context) {
	response.OK(c, gin.H{
		"smtp_host": h.getSetting("smtp_host"), "smtp_port": h.getSetting("smtp_port"),
		"smtp_user": h.getSetting("smtp_user"), "smtp_password": "",
		"smtp_from": h.getSetting("smtp_from"), "smtp_from_name": h.getSetting("smtp_from_name"),
		"smtp_use_tls": h.getSetting("smtp_use_tls") == "true",
		"smtp_use_ssl": h.getSetting("smtp_use_ssl") != "false",
	})
}

func (h *SettingsHandler) SaveSmtp(c *gin.Context) {
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	for k, v := range req {
		// 密码留空（或旧版掩码）表示不修改，避免误存空值覆盖真实密码
		if k == "smtp_password" {
			if sv, ok := v.(string); ok && sv != "" && sv != "***" { h.setSetting(k, sv) }
			continue
		}
		if sv, ok := v.(string); ok { h.setSetting(k, sv) }
		if bv, ok := v.(bool); ok {
			if bv { h.setSetting(k, "true") } else { h.setSetting(k, "false") }
		}
	}
	response.OKMsg(c, "SMTP配置已保存")
}

func (h *SettingsHandler) GetTracking(c *gin.Context) {
	response.OK(c, gin.H{
		"kuaidi100_customer": h.getSetting("kuaidi100_customer"),
		"kuaidi100_key": "***", "kuaidi100_secret": "***",
	})
}

// TestSmtp 真实发送一封测试邮件到指定邮箱（?to_email=）
func (h *SettingsHandler) TestSmtp(c *gin.Context) {
	to := c.Query("to_email")
	if to == "" {
		response.Fail(c, 400, "请提供收件邮箱")
		return
	}
	if h.getSetting("smtp_host") == "" {
		response.Fail(c, 422, "请先配置 SMTP 服务器")
		return
	}
	body := "<p>这是一封来自 <b>QSL 卡片管理系统</b> 的测试邮件。</p>" +
		"<div style=\"background:#f8f7f2;border-left:3px solid #f5a623;padding:12px 16px;margin:14px 0;\">收到此邮件说明 SMTP 配置正确，SWL 审批通知、回寄地址发送、收卡通知等功能将可正常发信。</div>" +
		"<p style=\"margin-bottom:0;\">73！</p>"
	if err := sendMail(h.db, to, "QSL 卡片管理系统测试邮件", body); err != nil {
		response.Fail(c, 500, "发送失败: "+err.Error())
		return
	}
	response.OKMsg(c, "测试邮件已发送，请查收")
}

func (h *SettingsHandler) SaveTracking(c *gin.Context) {
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	for k, v := range req {
		if sv, ok := v.(string); ok && sv != "***" { h.setSetting(k, sv) }
	}
	response.OKMsg(c, "快递追踪配置已保存")
}

func (h *SettingsHandler) GetSite(c *gin.Context) {
	response.OK(c, gin.H{"site_name": h.getSetting("site_name"), "site_url": h.getSetting("site_url"), "notify_email": h.getSetting("notify_email"), "site_notice": h.getSetting("site_notice"), "return_address_id": h.getSetting("return_address_id")})
}

func (h *SettingsHandler) SaveSite(c *gin.Context) {
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	for k, v := range req {
		if sv, ok := v.(string); ok { h.setSetting(k, sv) }
	}
	response.OKMsg(c, "站点设置已保存")
}
