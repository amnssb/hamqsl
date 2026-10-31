package handler

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

/*
账号安全：改用户名、看绑定情况、一键下线所有设备。

这几件事都会动"谁能进这个后台"，所以统一一条规矩：
改完就 bumpTokenVersion，旧令牌立刻作废，然后给当前这台设备换新的。
不然会出现"手机已经改完密码，笔记本上还开着"的尴尬。

调用示例：

	GET    /api/auth/security      → 当前绑定情况
	POST   /api/auth/username      {"username":"ba4slt","password":"..."}
	POST   /api/auth/sessions/revoke {"password":"..."}
*/
type AccountHandler struct{ db *gorm.DB }

func NewAccountHandler(db *gorm.DB) *AccountHandler { return &AccountHandler{db: db} }

// 用户名规则：只放行普通字符，避免和 URL、日志格式打架
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{3,32}$`)

// 保留字：这些名字容易被误认成系统账号。admin 本身是默认账号，不在此列。
var reservedUsernames = map[string]bool{
	"root": true, "system": true, "qsl": true,
	"administrator": true, "null": true, "undefined": true,
}

// Security 账号安全总览。
func (h *AccountHandler) Security(c *gin.Context) {
	user, ok := currentUser(h.db, c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	var rows []model.Passkey
	h.db.Where("user_id = ?", user.ID).Order("id").Find(&rows)
	passkeys := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		passkeys = append(passkeys, gin.H{
			"id":           p.ID,
			"name":         p.Name,
			"backup_able":  p.BackupAble,
			"last_used_at": p.LastUsedAt,
			"created_at":   p.CreatedAt,
		})
	}

	// totp_enabled 只看动态码，不要把通行密钥混进来（前端两个开关是分开的）
	var totpOn int64
	h.db.Model(&model.TotpSecret{}).Where("user_id = ? AND enabled = ?", user.ID, true).Count(&totpOn)
	response.OK(c, gin.H{
		"username":      user.Username,
		"display_name":  user.DisplayName,
		"is_admin":      user.IsAdmin,
		"totp_enabled":  totpOn > 0,
		"recovery_left": recoveryLeft(h.db, user.ID),
		"passkeys":      passkeys,
		"has_password":  user.HashPassword != "",
	})
}

// ChangeUsername 改自己的用户名。改完旧令牌全失效，当前设备换新令牌，不会把自己踢出去。
func (h *AccountHandler) ChangeUsername(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	user, ok := currentUser(h.db, c)
	if !ok {
		response.Unauthorized(c)
		return
	}

	newName := strings.TrimSpace(req.Username)
	if newName == user.Username {
		response.Fail(c, 400, "新用户名与当前一致")
		return
	}
	if !usernamePattern.MatchString(newName) {
		response.Fail(c, 400, "用户名只能是 3-32 位字母、数字、下划线、点或连字符")
		return
	}
	if reservedUsernames[strings.ToLower(newName)] {
		response.Fail(c, 400, "该用户名已被系统保留")
		return
	}
	// 有口令就要求确认；纯通行密钥账号靠当前会话（它是通行密钥验过的）即可
	if user.HashPassword != "" && !middleware.CheckPassword(user.HashPassword, req.Password) {
		response.Fail(c, 400, "口令不正确")
		return
	}

	var dup int64
	h.db.Model(&model.User{}).Where("username = ?", newName).Count(&dup)
	if dup > 0 {
		response.Fail(c, 409, "该用户名已被占用")
		return
	}

	if err := h.db.Model(&model.User{}).Where("id = ?", user.ID).Update("username", newName).Error; err != nil {
		response.Error(c, "保存失败")
		return
	}
	writeAudit(h.db, "USERNAME_CHANGED", newName, "用户名由 "+user.Username+" 改为 "+newName, middleware.RealClientIP(c))

	// 通行密钥不依赖用户名（靠 credential_id 关联），改名后照常可用
	bumpTokenVersion(h.db, &user)
	user.Username = newName
	issueTokens(c, &user, amrOf(c))
}

// RevokeSessions 让该账号在所有设备上的旧令牌失效，当前设备换新的继续用。
func (h *AccountHandler) RevokeSessions(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	user, ok := currentUser(h.db, c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	if user.HashPassword != "" && !middleware.CheckPassword(user.HashPassword, req.Password) {
		response.Fail(c, 400, "口令不正确")
		return
	}

	bumpTokenVersion(h.db, &user)
	if err := h.db.Where("user_id = ?", user.ID).Delete(&model.LoginTicket{}).Error; err != nil {
		response.Error(c, "清理登录票据失败")
		return
	}
	writeAudit(h.db, "SESSIONS_REVOKED", user.Username, "强制下线所有设备", middleware.RealClientIP(c))
	issueTokens(c, &user, amrOf(c))
}
