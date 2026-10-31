package handler

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"

	"qsl-management/internal/config"
	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

/*
二步验证（TOTP）：生成密钥、确认启用、校验验证码、恢复码。

密钥用 AES-256-GCM 加密后落库，主密钥在 data/.mfa_key；恢复码用 HMAC-SHA256 存储。
两者都不用 bcrypt：恢复码本身是 50 位随机量，密钥在手才能验，离线爆破没意义，而 bcrypt 一次 60ms，
10 个码逐个比就是 600ms，登录时太慢。

调用示例：

	th := NewTotpHandler(db)
	th.Setup(c)                        // POST /api/auth/totp/setup
	ok := verifyTotpCode(db, user, "123456")
	codes := newRecoveryCodes()
*/
const (
	totpIssuer    = "QSL 管理台"
	totpPeriod    = 30 // 秒，RFC 6238 默认值，不要再改：改了这个所有已绑定的验证器全失效
	totpDigits    = otp.DigitsSix
	totpSkew      = 1 // 容忍前后各一个窗口（±30 秒），手机时钟差点也能用
	recoveryCount = 10
)

type TotpHandler struct{ db *gorm.DB }

func NewTotpHandler(db *gorm.DB) *TotpHandler { return &TotpHandler{db: db} }

// --- 加解密 ---

// sealSecret 用主密钥加密 TOTP secret。
func sealSecret(plain string) ([]byte, error) {
	block, err := aes.NewCipher(config.MFAKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(plain), nil), nil
}

func openSecret(blob []byte) (string, error) {
	block, err := aes.NewCipher(config.MFAKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(blob) < gcm.NonceSize() {
		return "", errors.New("密钥数据损坏")
	}
	nonce, ciphertext := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// --- 恢复码 ---

// newRecoveryCodes 生成 10 个一次性恢复码，形如 A2C4E-7H9JK。
func newRecoveryCodes() []string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 去掉了容易看错的 I O 0 1
	codes := make([]string, 0, recoveryCount)
	for i := 0; i < recoveryCount; i++ {
		buf := make([]byte, 10)
		if _, err := rand.Read(buf); err != nil {
			return nil
		}
		var sb strings.Builder
		for j, b := range buf {
			if j == 5 {
				sb.WriteByte('-')
			}
			sb.WriteByte(alphabet[int(b)%len(alphabet)])
		}
		codes = append(codes, sb.String())
	}
	return codes
}

// hashRecoveryCode 用主密钥做 HMAC。没有 .mfa_key 就算拿到数据库也验不出码。
func hashRecoveryCode(code string) string {
	mac := hmac.New(sha256.New, config.MFAKey())
	mac.Write([]byte(normalizeCode(code)))
	return hex.EncodeToString(mac.Sum(nil))
}

// normalizeCode 统一大小写并去掉分隔符，用户输 a2c4e7h9jk 也能过。
func normalizeCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
}

// --- 校验 ---

// loadSecret 取出某个账号的已启用密钥；未绑定或未启用返回空串。
func loadSecret(db *gorm.DB, userID uint) (*model.TotpSecret, string) {
	var s model.TotpSecret
	if err := db.Where("user_id = ?", userID).First(&s).Error; err != nil {
		return nil, ""
	}
	if !s.Enabled {
		return &s, ""
	}
	plain, err := openSecret(s.SecretEnc)
	if err != nil {
		return &s, ""
	}
	return &s, plain
}

// verifyTotpCode 校验一次动态码，并挡掉同一时间窗内的重放。
func verifyTotpCode(db *gorm.DB, user *model.User, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	s, secret := loadSecret(db, user.ID)
	if s == nil || secret == "" {
		return false
	}

	step := time.Now().Unix() / totpPeriod
	if s.LastUsedStep == step {
		return false // 这一步已经用过了，同码重放
	}

	ok, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      totpSkew,
		Digits:    totpDigits,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !ok {
		return false
	}

	db.Model(&model.TotpSecret{}).Where("user_id = ?", user.ID).Update("last_used_step", step)
	return true
}

// useRecoveryCode 校验并作废一个恢复码。
func useRecoveryCode(db *gorm.DB, user *model.User, code string) bool {
	var s model.TotpSecret
	if err := db.Where("user_id = ?", user.ID).First(&s).Error; err != nil {
		return false
	}
	var hashes []string
	if err := json.Unmarshal([]byte(s.RecoveryHash), &hashes); err != nil {
		return false
	}

	target := hashRecoveryCode(code)
	kept := make([]string, 0, len(hashes))
	matched := false
	for _, h := range hashes {
		if !matched && hmac.Equal([]byte(h), []byte(target)) {
			matched = true
			continue // 用掉就丢
		}
		kept = append(kept, h)
	}
	if !matched {
		return false
	}

	encoded, _ := json.Marshal(kept)
	db.Model(&model.TotpSecret{}).Where("user_id = ?", user.ID).Update("recovery_hash", string(encoded))
	return true
}

// recoveryLeft 剩余可用恢复码数量，账号安全页展示用。
func recoveryLeft(db *gorm.DB, userID uint) int {
	var s model.TotpSecret
	if err := db.Where("user_id = ?", userID).First(&s).Error; err != nil {
		return 0
	}
	var hashes []string
	if err := json.Unmarshal([]byte(s.RecoveryHash), &hashes); err != nil {
		return 0
	}
	return len(hashes)
}

// --- 接口 ---

// Setup 生成新的 TOTP 密钥，但先不启用。
// 必须让用户拿验证码回来确认（Enable），否则手机还没添加上就先把自己锁死了。
func (h *TotpHandler) Setup(c *gin.Context) {
	user, ok := currentUser(h.db, c)
	if !ok {
		response.Unauthorized(c)
		return
	}
	var existing model.TotpSecret
	if err := h.db.Where("user_id = ?", user.ID).First(&existing).Error; err == nil && existing.Enabled {
		response.Fail(c, 400, "二步验证已启用，如需更换请先关闭")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{Issuer: totpIssuer, AccountName: user.Username})
	if err != nil {
		response.Error(c, "生成密钥失败")
		return
	}
	enc, err := sealSecret(key.Secret())
	if err != nil {
		response.Error(c, "生成密钥失败")
		return
	}

	row := model.TotpSecret{UserID: user.ID, SecretEnc: enc, Enabled: false, RecoveryHash: "[]", CreatedAt: cstNow()}
	if err := h.db.Where("user_id = ?", user.ID).Delete(&model.TotpSecret{}).Error; err != nil {
		response.Error(c, "保存失败")
		return
	}
	if err := h.db.Create(&row).Error; err != nil {
		response.Error(c, "保存失败")
		return
	}

	response.OK(c, gin.H{"secret": key.Secret(), "otpauth_url": key.URL()})
}

// Enable 用用户输入的第一个验证码确认绑定，并一次性发放恢复码。
func (h *TotpHandler) Enable(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
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

	var row model.TotpSecret
	if err := h.db.Where("user_id = ?", user.ID).First(&row).Error; err != nil {
		response.Fail(c, 400, "请先生成密钥")
		return
	}
	if row.Enabled {
		response.Fail(c, 400, "二步验证已启用")
		return
	}
	secret, err := openSecret(row.SecretEnc)
	if err != nil {
		response.Error(c, "密钥数据损坏，请重新生成")
		return
	}
	ok2, verr := totp.ValidateCustom(strings.TrimSpace(req.Code), secret, time.Now().UTC(), totp.ValidateOpts{
		Period: totpPeriod, Skew: totpSkew, Digits: totpDigits, Algorithm: otp.AlgorithmSHA1,
	})
	if verr != nil || !ok2 {
		response.Fail(c, 400, "验证码不正确")
		return
	}

	codes := newRecoveryCodes()
	hashes := make([]string, 0, len(codes))
	for _, code := range codes {
		hashes = append(hashes, hashRecoveryCode(code))
	}
	encoded, _ := json.Marshal(hashes)

	now := cstNow()
	if err := h.db.Model(&model.TotpSecret{}).Where("user_id = ?", user.ID).Updates(map[string]any{
		"enabled":        true,
		"recovery_hash":  string(encoded),
		"confirmed_at":   now,
		"last_used_step": time.Now().Unix() / totpPeriod, // 启用时用掉的那个码不能再用于登录
	}).Error; err != nil {
		response.Error(c, "保存失败")
		return
	}

	writeAudit(h.db, "MFA_ENABLED", user.Username, "启用二步验证", middleware.RealClientIP(c))
	// 明文恢复码只在此刻返回这一次，之后库里只有哈希
	response.OK(c, gin.H{"recovery_codes": codes})
}

// Disable 关闭二步验证：既要口令也要当前验证码，防手滑也防被人摸到会话就关。
func (h *TotpHandler) Disable(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
		Code     string `json:"code"`
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
	if !middleware.CheckPassword(user.HashPassword, req.Password) {
		response.Fail(c, 400, "口令不正确")
		return
	}
	if !verifyTotpCode(h.db, &user, req.Code) {
		response.Fail(c, 400, "验证码不正确")
		return
	}

	if err := h.db.Where("user_id = ?", user.ID).Delete(&model.TotpSecret{}).Error; err != nil {
		response.Error(c, "保存失败")
		return
	}
	writeAudit(h.db, "MFA_DISABLED", user.Username, "关闭二步验证", middleware.RealClientIP(c))

	// 少了一道因子，旧令牌一律作废，然后给当前这台设备换新的
	bumpTokenVersion(h.db, &user)
	issueTokens(c, &user, amrOf(c))
}

// ResetSecondFactor 逃生用：清空某个账号的全部第二因子与失败计数。
// 只在 ADMIN_RESET=1 启动时调用，配合一次性随机口令，是丢了设备之后的唯一活路。
func ResetSecondFactor(db *gorm.DB, username string) error {
	var user model.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return fmt.Errorf("账号 %s 不存在", username)
	}
	if err := db.Where("user_id = ?", user.ID).Delete(&model.TotpSecret{}).Error; err != nil {
		return err
	}
	if err := db.Where("user_id = ?", user.ID).Delete(&model.Passkey{}).Error; err != nil {
		return err
	}
	if err := db.Where("user_id = ?", user.ID).Delete(&model.LoginTicket{}).Error; err != nil {
		return err
	}
	return db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"failed_count":  0,
		"locked_until":  time.Time{},
		"token_version": user.TokenVersion + 1,
	}).Error
}
