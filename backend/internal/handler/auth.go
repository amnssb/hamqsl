package handler

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

/*
登录与账号。

登录现在是三岔口：口令错 → 401；口令对且没绑第二因子 → 直接发令牌；
口令对且绑了 → 只发一张票据，等第二因子过了再发令牌。
这样"知道口令"和"能进后台"就分开了。

调用示例：

	POST /api/auth/login         {"username":"admin","password":"..."}
	POST /api/auth/mfa/totp      {"ticket":"...","code":"123456"}
	POST /api/auth/mfa/recovery  {"ticket":"...","code":"ABCDE-FGHJK"}
*/
type AuthHandler struct{ db *gorm.DB }

func NewAuthHandler(db *gorm.DB) *AuthHandler { return &AuthHandler{db: db} }

// writeAudit 写入审计日志（失败仅记服务端日志，不影响主流程）。
// IP 必须用 RealClientIP：用 c.ClientIP() 记下来的日志，攻击者想写什么就写什么。
func writeAudit(db *gorm.DB, action, operator, detail, ip string) {
	entry := model.AuditLog{
		Action:       action,
		ResourceType: "auth",
		Operator:     operator,
		Detail:       detail,
		IPAddress:    ip,
		CreatedAt:    cstNow(),
	}
	if err := db.Create(&entry).Error; err != nil {
		log.Printf("[audit] 写入失败 action=%s err=%v", action, err)
	}
}

// --- 小工具 ---

// currentUser 取当前登录账号（已由 AuthRequired 确认存在且启用）。
func currentUser(db *gorm.DB, c *gin.Context) (model.User, bool) {
	var user model.User
	if err := db.Where("username = ?", c.GetString("username")).First(&user).Error; err != nil {
		return user, false
	}
	return user, true
}

// amrOf 取当前会话的认证方式，换发令牌时用它保持同等强度。
func amrOf(c *gin.Context) []string {
	if v, ok := c.Get("amr"); ok {
		if list, ok := v.([]string); ok && len(list) > 0 {
			return list
		}
	}
	return []string{"pwd"}
}

// issueTokens 签发并返回令牌对。
func issueTokens(c *gin.Context, user *model.User, amr []string) {
	access, refresh, err := middleware.GenerateToken(user.Username, user.TokenVersion, amr)
	if err != nil {
		response.Error(c, "生成令牌失败")
		return
	}
	response.OK(c, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "bearer",
		"expires_in":    middleware.AccessTTL(),
	})
}

// bumpTokenVersion 让该账号所有已签发令牌立刻失效。
func bumpTokenVersion(db *gorm.DB, user *model.User) {
	user.TokenVersion++
	db.Model(&model.User{}).Where("id = ?", user.ID).Update("token_version", user.TokenVersion)
}

// secondFactorMethods 账号绑了哪些第二因子，登录时用来告诉前端该显示哪几个选项。
// 做成包级函数是因为"通行密钥"那边判断"还能不能删"时也要用同一套口径。
func secondFactorMethods(db *gorm.DB, userID uint) (bool, []string) {
	var n int64
	db.Model(&model.TotpSecret{}).Where("user_id = ? AND enabled = ?", userID, true).Count(&n)
	methods := []string{}
	if n > 0 {
		methods = append(methods, "totp")
	}
	db.Model(&model.Passkey{}).Where("user_id = ?", userID).Count(&n)
	if n > 0 {
		methods = append(methods, "passkey")
	}
	return len(methods) > 0, methods
}

// recordLoginFail 记一次失败，连续 5 次锁定 15 分钟。
// 做成包级函数，"通行密钥"那条路也要记在同一份计数上，不能各算一套。
func recordLoginFail(db *gorm.DB, user *model.User) {
	user.FailedCount++
	if user.FailedCount >= 5 {
		user.LockedUntil = time.Now().Add(15 * time.Minute)
		user.FailedCount = 0
	}
	db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"failed_count": user.FailedCount,
		"locked_until": user.LockedUntil,
	})
}

// clearLoginFail 登录成功后清零失败计数并记下登录时间。
func clearLoginFail(db *gorm.DB, user *model.User) {
	now := time.Now()
	db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"failed_count":  0,
		"locked_until":  time.Time{},
		"last_login_at": now,
	})
	user.FailedCount = 0
	user.LockedUntil = time.Time{}
	user.LastLoginAt = &now
}

// --- 登录 ---

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	ip := middleware.RealClientIP(c)
	username := strings.TrimSpace(req.Username)

	// 账号维度配额。伪造 X-Forwarded-For 只能换掉中间件里的 IP 桶，这一桶与 IP 无关。
	if !middleware.AllowLoginAttempt(username) {
		writeAudit(h.db, "LOGIN_RATE_LIMITED", username, "账号维度限速", ip)
		c.Header("Retry-After", "300")
		response.Fail(c, 429, "登录尝试过于频繁，请稍后再试")
		return
	}

	var user model.User
	err := h.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		// 用户不存在也跑一次等价 bcrypt：否则快慢路径就是一个免费的账号枚举信道
		middleware.BurnPasswordTime(req.Password)
		writeAudit(h.db, "LOGIN_FAILED", username, "用户不存在", ip)
		response.Unauthorized(c)
		return
	}
	if !user.IsActive {
		middleware.BurnPasswordTime(req.Password)
		writeAudit(h.db, "LOGIN_FAILED", username, "账号已停用", ip)
		response.Unauthorized(c)
		return
	}
	if user.LockedUntil.After(time.Now()) {
		writeAudit(h.db, "LOGIN_FAILED", username, "账号锁定中", ip)
		response.Fail(c, 423, "账号已临时锁定，请稍后再试")
		return
	}
	// 纯通行密钥账号没有口令可比，同样烧掉一次 bcrypt 保持耗时一致
	if user.HashPassword == "" {
		middleware.BurnPasswordTime(req.Password)
		writeAudit(h.db, "LOGIN_FAILED", username, "该账号未设置口令", ip)
		response.Unauthorized(c)
		return
	}
	if !middleware.CheckPassword(user.HashPassword, req.Password) {
		recordLoginFail(h.db, &user)
		writeAudit(h.db, "LOGIN_FAILED", username, "口令错误", ip)
		response.Unauthorized(c)
		return
	}

	// 口令正确
	clearLoginFail(h.db, &user)
	middleware.ResetLoginAttempt(username)

	needMFA, methods := secondFactorMethods(h.db, user.ID)
	if needMFA {
		ticket, terr := issueTicket(h.db, user.ID, "mfa", "", ip)
		if terr != nil {
			response.Error(c, "生成登录票据失败")
			return
		}
		response.OK(c, gin.H{
			"mfa_required": true,
			"ticket":       ticket,
			"methods":      methods,
			"expires_in":   int(ticketTTL.Seconds()),
		})
		return
	}

	writeAudit(h.db, "LOGIN_OK", user.Username, "登录成功", ip)
	issueTokens(c, &user, []string{"pwd"})
}

// MfaTotp 用动态码完成第二因子。
func (h *AuthHandler) MfaTotp(c *gin.Context) {
	var req struct {
		Ticket string `json:"ticket"`
		Code   string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	// 先只看不消费：验证码打错不该把票据烧掉，用户还得重输一遍口令
	tk, err := peekTicket(h.db, req.Ticket, "mfa")
	if err != nil {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}
	var user model.User
	if err := h.db.First(&user, tk.UserID).Error; err != nil || !user.IsActive {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}
	if !middleware.AllowTotpAttempt(user.Username) {
		response.Fail(c, 429, "验证码尝试过于频繁，请稍后再试")
		return
	}
	if !verifyTotpCode(h.db, &user, req.Code) {
		writeAudit(h.db, "MFA_FAILED", user.Username, "动态码不正确", middleware.RealClientIP(c))
		response.Unauthorized(c)
		return
	}
	if _, err := consumeTicket(h.db, req.Ticket, "mfa"); err != nil {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}

	clearLoginFail(h.db, &user)
	writeAudit(h.db, "LOGIN_OK", user.Username, "口令 + 动态码登录", middleware.RealClientIP(c))
	issueTokens(c, &user, []string{"pwd", "totp"})
}

// MfaRecovery 用一次性恢复码完成第二因子（手机丢了、验证器没了）。
func (h *AuthHandler) MfaRecovery(c *gin.Context) {
	var req struct {
		Ticket string `json:"ticket"`
		Code   string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}

	tk, err := peekTicket(h.db, req.Ticket, "mfa")
	if err != nil {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}
	var user model.User
	if err := h.db.First(&user, tk.UserID).Error; err != nil || !user.IsActive {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}
	if !middleware.AllowTotpAttempt(user.Username) {
		response.Fail(c, 429, "尝试过于频繁，请稍后再试")
		return
	}
	if !useRecoveryCode(h.db, &user, req.Code) {
		writeAudit(h.db, "MFA_FAILED", user.Username, "恢复码不正确", middleware.RealClientIP(c))
		response.Unauthorized(c)
		return
	}
	if _, err := consumeTicket(h.db, req.Ticket, "mfa"); err != nil {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}

	clearLoginFail(h.db, &user)
	writeAudit(h.db, "LOGIN_OK", user.Username, "恢复码登录", middleware.RealClientIP(c))
	issueTokens(c, &user, []string{"pwd", "totp"})
}

// Me 当前账号信息。
func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := currentUser(h.db, c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	mfaOn, _ := secondFactorMethods(h.db, user.ID)
	var passkeys int64
	h.db.Model(&model.Passkey{}).Where("user_id = ?", user.ID).Count(&passkeys)
	response.OK(c, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"is_admin":     user.IsAdmin,
		"mfa_enabled":  mfaOn,
		"passkeys":     passkeys,
	})
}

// ChangePassword 当前登录用户修改自己的密码（需原密码验证）。
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	if strings.TrimSpace(req.OldPassword) == "" {
		response.Fail(c, 400, "请输入原密码")
		return
	}
	if len(req.NewPassword) < 8 {
		response.Fail(c, 400, "新密码至少 8 位")
		return
	}
	user, ok := currentUser(h.db, c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	if !middleware.CheckPassword(user.HashPassword, req.OldPassword) {
		response.Fail(c, 400, "原密码错误")
		return
	}

	hash, err := middleware.HashPassword(req.NewPassword)
	if err != nil {
		response.Error(c, "密码加密失败")
		return
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", user.ID).Update("hash_password", hash).Error; err != nil {
		response.Error(c, "保存失败，请稍后重试")
		return
	}
	writeAudit(h.db, "CHANGE_PASSWORD", user.Username, "修改密码成功", middleware.RealClientIP(c))

	// 改密后所有旧令牌作废，包括当前这个 —— 前端会跳回登录页
	bumpTokenVersion(h.db, &user)
	response.OKMsg(c, "密码已修改，请重新登录")
}

// Refresh 用刷新令牌换新的令牌对。
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		response.Fail(c, 400, "缺少刷新令牌")
		return
	}

	claims, err := middleware.ParseRefreshToken(strings.TrimSpace(req.RefreshToken))
	if err != nil {
		response.Fail(c, 401, "无效的刷新令牌")
		return
	}
	var user model.User
	if err := h.db.Where("username = ?", claims.Subject).First(&user).Error; err != nil || !user.IsActive {
		response.Fail(c, 401, "无效的刷新令牌")
		return
	}
	// 版本对不上说明期间改过密 / 改过名 / 解绑过第二因子，旧凭据一律不认
	if claims.Ver != user.TokenVersion {
		response.Fail(c, 401, "凭据已失效，请重新登录")
		return
	}

	// 保持原来的认证强度：用 passkey 登的，刷新出来的还是 passkey 级别
	amr := claims.AMR
	if len(amr) == 0 {
		amr = []string{"pwd"}
	}
	issueTokens(c, &user, amr)
}
