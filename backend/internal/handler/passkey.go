package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"

	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

/*
通行密钥（WebAuthn）。

两条路：
 1. 免密直接登录 —— 浏览器弹账号选择器，服务器一开始并不知道来的是谁（discoverable credential）；
 2. 作为第二因子 —— 口令已经过了，再用通行密钥签一次名。

硬约束：通行密钥绑定域名（RP ID），只认 HTTPS。
明文 HTTP 或换了域名，已有的密钥一律失效 —— 这是 WebAuthn 的设计，不是 bug。

调用示例：

	ph := NewPasskeyHandler(db, wa)   // wa 为 nil 时整个功能关闭
	POST /api/auth/passkey/login/begin
	POST /api/auth/passkey/login/finish?ticket=...
*/
type PasskeyHandler struct {
	db *gorm.DB
	wa *webauthn.WebAuthn // nil 表示未配置 WEBAUTHN_RP_ID，功能关闭
}

func NewPasskeyHandler(db *gorm.DB, wa *webauthn.WebAuthn) *PasskeyHandler {
	return &PasskeyHandler{db: db, wa: wa}
}

// Enabled 是否可用。未配置域名时前端不显示按钮。
func (h *PasskeyHandler) Enabled() bool { return h.wa != nil }

// --- WebAuthn 用户适配 ---

// passkeyUser 把数据库里的账号包装成库需要的 User 接口。
type passkeyUser struct {
	user  model.User
	creds []webauthn.Credential
}

func (u *passkeyUser) WebAuthnID() []byte {
	return []byte(strconv.FormatUint(uint64(u.user.ID), 10))
}

func (u *passkeyUser) WebAuthnName() string { return u.user.Username }

func (u *passkeyUser) WebAuthnDisplayName() string {
	if strings.TrimSpace(u.user.DisplayName) != "" {
		return u.user.DisplayName
	}
	return u.user.Username
}

func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

// toCredential 数据库行 → 库里的凭据结构。
func toCredential(p model.Passkey) webauthn.Credential {
	var transports []protocol.AuthenticatorTransport
	for _, t := range strings.Split(p.Transports, ",") {
		if t = strings.TrimSpace(t); t != "" {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}
	}
	return webauthn.Credential{
		ID:              p.CredentialID,
		PublicKey:       p.PublicKey,
		AttestationType: p.Attestation,
		Transport:       transports,
		Flags:           webauthn.CredentialFlags{BackupEligible: p.BackupAble, BackupState: p.BackupAble},
		Authenticator:   webauthn.Authenticator{SignCount: p.SignCount, AAGUID: p.AAGUID},
	}
}

// loadPasskeyUser 连账号带密钥一起读出来。
func loadPasskeyUser(db *gorm.DB, userID uint) (*passkeyUser, error) {
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	var rows []model.Passkey
	db.Where("user_id = ?", userID).Find(&rows)
	creds := make([]webauthn.Credential, 0, len(rows))
	for _, p := range rows {
		creds = append(creds, toCredential(p))
	}
	return &passkeyUser{user: user, creds: creds}, nil
}

// --- 注册（绑定新密钥） ---

func (h *PasskeyHandler) RegisterBegin(c *gin.Context) {
	if !h.Enabled() {
		response.Fail(c, 400, "本机未启用通行密钥（缺少 WEBAUTHN_RP_ID 配置）")
		return
	}
	user, ok := currentUser(h.db, c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req) // 名字可选，缺省叫"通行密钥"

	pu, err := loadPasskeyUser(h.db, user.ID)
	if err != nil {
		response.Error(c, "读取账号失败")
		return
	}
	creation, session, err := h.wa.BeginRegistration(pu)
	if err != nil {
		response.Fail(c, 400, "无法开始注册："+err.Error())
		return
	}

	// 中间态存进票据表：challenge 必须一次性、5 分钟内有效
	raw, _ := json.Marshal(session)
	ticket, err := issueTicket(h.db, user.ID, "register", string(raw), middleware.RealClientIP(c))
	if err != nil {
		response.Error(c, "生成注册票据失败")
		return
	}
	response.OK(c, gin.H{"publicKey": creation.Response, "ticket": ticket})
}

func (h *PasskeyHandler) RegisterFinish(c *gin.Context) {
	if !h.Enabled() {
		response.Fail(c, 400, "本机未启用通行密钥")
		return
	}
	tk, err := peekTicket(h.db, c.Query("ticket"), "register")
	if err != nil {
		response.Fail(c, 410, "注册已超时，请重试")
		return
	}
	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		name = "通行密钥"
	}

	var session webauthn.SessionData
	if err := json.Unmarshal([]byte(tk.SessionData), &session); err != nil {
		response.Fail(c, 410, "注册已超时，请重试")
		return
	}
	pu, err := loadPasskeyUser(h.db, tk.UserID)
	if err != nil {
		response.Fail(c, 410, "注册已超时，请重试")
		return
	}

	cred, err := h.wa.FinishRegistration(pu, session, c.Request)
	if err != nil {
		response.Fail(c, 400, "注册失败："+err.Error())
		return
	}
	if _, err := consumeTicket(h.db, tk.ID, "register"); err != nil {
		response.Fail(c, 410, "注册已超时，请重试")
		return
	}

	transports := make([]string, 0, len(cred.Transport))
	for _, t := range cred.Transport {
		transports = append(transports, string(t))
	}
	row := model.Passkey{
		UserID:       pu.user.ID,
		CredentialID: cred.ID,
		PublicKey:    cred.PublicKey,
		Attestation:  cred.AttestationType,
		SignCount:    cred.Authenticator.SignCount,
		Transports:   strings.Join(transports, ","),
		AAGUID:       cred.Authenticator.AAGUID,
		BackupAble:   cred.Flags.BackupEligible,
		Name:         name,
		CreatedAt:    cstNow(),
	}
	if err := h.db.Create(&row).Error; err != nil {
		response.Error(c, "保存通行密钥失败")
		return
	}

	writeAudit(h.db, "PASSKEY_ADDED", pu.user.Username, "新增通行密钥："+name, middleware.RealClientIP(c))
	response.OK(c, gin.H{"id": row.ID, "name": row.Name})
}

// --- 免密登录 ---

func (h *PasskeyHandler) LoginBegin(c *gin.Context) {
	if !h.Enabled() {
		response.Fail(c, 400, "本机未启用通行密钥")
		return
	}
	assertion, session, err := h.wa.BeginDiscoverableLogin()
	if err != nil {
		response.Fail(c, 400, "无法开始登录："+err.Error())
		return
	}
	raw, _ := json.Marshal(session)
	ticket, err := issueTicket(h.db, 0, "login", string(raw), middleware.RealClientIP(c))
	if err != nil {
		response.Error(c, "生成登录票据失败")
		return
	}
	response.OK(c, gin.H{"publicKey": assertion.Response, "ticket": ticket})
}

func (h *PasskeyHandler) LoginFinish(c *gin.Context) {
	if !h.Enabled() {
		response.Fail(c, 400, "本机未启用通行密钥")
		return
	}
	tk, err := peekTicket(h.db, c.Query("ticket"), "login")
	if err != nil {
		response.Fail(c, 410, "登录已超时，请重新开始")
		return
	}
	var session webauthn.SessionData
	if err := json.Unmarshal([]byte(tk.SessionData), &session); err != nil {
		response.Fail(c, 410, "登录已超时，请重新开始")
		return
	}

	// 库里不知道来的是哪把钥匙，靠 rawID 反查归属
	lookup := func(rawID, userHandle []byte) (webauthn.User, error) {
		var row model.Passkey
		if err := h.db.Where("credential_id = ?", rawID).First(&row).Error; err != nil {
			return nil, errors.New("未登记过的通行密钥")
		}
		return loadPasskeyUser(h.db, row.UserID)
	}
	cred, err := h.wa.FinishDiscoverableLogin(lookup, session, c.Request)
	if err != nil {
		writeAudit(h.db, "PASSKEY_FAILED", "-", "免密登录失败："+err.Error(), middleware.RealClientIP(c))
		response.Unauthorized(c)
		return
	}
	if _, err := consumeTicket(h.db, tk.ID, "login"); err != nil {
		response.Fail(c, 410, "登录已超时，请重新开始")
		return
	}

	var row model.Passkey
	if err := h.db.Where("credential_id = ?", cred.ID).First(&row).Error; err != nil {
		response.Unauthorized(c)
		return
	}
	user, err := h.touchPasskey(&row, cred)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	clearLoginFail(h.db, &user)
	writeAudit(h.db, "LOGIN_OK", user.Username, "通行密钥免密登录", middleware.RealClientIP(c))
	issueTokens(c, &user, []string{"passkey"})
}

// --- 作为第二因子 ---

func (h *PasskeyHandler) MfaBegin(c *gin.Context) {
	if !h.Enabled() {
		response.Fail(c, 400, "本机未启用通行密钥")
		return
	}
	var req struct {
		Ticket string `json:"ticket"`
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
	pu, err := loadPasskeyUser(h.db, tk.UserID)
	if err != nil || len(pu.creds) == 0 {
		response.Fail(c, 400, "该账号没有绑定通行密钥")
		return
	}

	assertion, session, err := h.wa.BeginLogin(pu)
	if err != nil {
		response.Fail(c, 400, "无法开始验证："+err.Error())
		return
	}
	// 中间态写回同一张票据，finish 时一起核销
	raw, _ := json.Marshal(session)
	if err := h.db.Model(&model.LoginTicket{}).Where("id = ?", tk.ID).Update("session_data", string(raw)).Error; err != nil {
		response.Error(c, "生成验证票据失败")
		return
	}
	response.OK(c, gin.H{"publicKey": assertion.Response, "ticket": tk.ID})
}

func (h *PasskeyHandler) MfaFinish(c *gin.Context) {
	if !h.Enabled() {
		response.Fail(c, 400, "本机未启用通行密钥")
		return
	}
	tk, err := peekTicket(h.db, c.Query("ticket"), "mfa")
	if err != nil {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}
	var session webauthn.SessionData
	if err := json.Unmarshal([]byte(tk.SessionData), &session); err != nil {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}
	pu, err := loadPasskeyUser(h.db, tk.UserID)
	if err != nil {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}

	cred, err := h.wa.FinishLogin(pu, session, c.Request)
	if err != nil {
		writeAudit(h.db, "MFA_FAILED", pu.user.Username, "通行密钥验证失败："+err.Error(), middleware.RealClientIP(c))
		response.Unauthorized(c)
		return
	}
	if _, err := consumeTicket(h.db, tk.ID, "mfa"); err != nil {
		response.Fail(c, 410, "登录已超时，请重新登录")
		return
	}

	var row model.Passkey
	if err := h.db.Where("credential_id = ?", cred.ID).First(&row).Error; err != nil {
		response.Unauthorized(c)
		return
	}
	user, err := h.touchPasskey(&row, cred)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	clearLoginFail(h.db, &user)
	writeAudit(h.db, "LOGIN_OK", user.Username, "口令 + 通行密钥登录", middleware.RealClientIP(c))
	issueTokens(c, &user, []string{"pwd", "passkey"})
}

// touchPasskey 更新签名计数器与最后使用时间，并做克隆检测。
//
// 只有库里的计数和这次的计数都大于 0 才比较：云同步型 passkey 常年是 0，
// 硬比会把正常用户误判成克隆。
func (h *PasskeyHandler) touchPasskey(row *model.Passkey, cred *webauthn.Credential) (model.User, error) {
	newCount := cred.Authenticator.SignCount
	if row.SignCount > 0 && newCount > 0 && newCount <= row.SignCount {
		h.db.Model(&model.Passkey{}).Where("id = ?", row.ID).Update("clone_warned", true)
		writeAudit(h.db, "PASSKEY_COUNT_ANOMALY", "-", "签名计数器未增长，疑似密钥被复制", "")
		return model.User{}, errors.New("凭据异常")
	}

	var user model.User
	if err := h.db.First(&user, row.UserID).Error; err != nil || !user.IsActive {
		return user, errors.New("账号不可用")
	}

	now := cstNow()
	h.db.Model(&model.Passkey{}).Where("id = ?", row.ID).Updates(map[string]any{
		"sign_count":   newCount,
		"last_used_at": now,
	})
	return user, nil
}

// Remove 解绑一把通行密钥。要口令确认，且不允许把最后一条登录路径删掉。
func (h *PasskeyHandler) Remove(c *gin.Context) {
	user, ok := currentUser(h.db, c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	if !middleware.CheckPassword(user.HashPassword, req.Password) {
		response.Fail(c, 400, "口令不正确")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	var row model.Passkey
	if err := h.db.Where("id = ? AND user_id = ?", id, user.ID).First(&row).Error; err != nil {
		response.NotFound(c, "通行密钥不存在")
		return
	}

	// 删掉之后至少还要留一条能进门的钥匙：口令、另一个通行密钥、或已启用的动态码
	var left, totpOn int64
	h.db.Model(&model.Passkey{}).Where("user_id = ? AND id <> ?", user.ID, row.ID).Count(&left)
	h.db.Model(&model.TotpSecret{}).Where("user_id = ? AND enabled = ?", user.ID, true).Count(&totpOn)
	if left == 0 && totpOn == 0 && user.HashPassword == "" {
		response.Fail(c, 400, "这是最后一条登录方式，删除后将无法登录")
		return
	}

	if err := h.db.Delete(&model.Passkey{}, row.ID).Error; err != nil {
		response.Error(c, "删除失败")
		return
	}
	writeAudit(h.db, "PASSKEY_REMOVED", user.Username, "解绑通行密钥："+row.Name, middleware.RealClientIP(c))

	bumpTokenVersion(h.db, &user)
	issueTokens(c, &user, amrOf(c))
}
