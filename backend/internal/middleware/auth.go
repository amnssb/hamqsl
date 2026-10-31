package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"qsl-management/internal/model"
)

/*
令牌与鉴权。

令牌里除了用户名，还带两样东西：

	ver —— 账号的令牌版本。改动安全设置后账号的 TokenVersion 加一，旧令牌立刻作废。
	amr —— 这次登录用了哪些方式（pwd / totp / passkey），敏感接口靠它判断是否已过二次验证。

调用示例：

	access, refresh, err := middleware.GenerateToken(user.Username, user.TokenVersion, []string{"pwd", "totp"})
	r.Use(middleware.AuthRequired(db))
*/
const (
	accessTTL   = 15 * time.Minute
	refreshTTL  = 7 * 24 * time.Hour
	tokenIssuer = "qsl-management"
)

var jwtSecret []byte

func SetSecret(secret string) { jwtSecret = []byte(secret) }

// AccessTTL 访问令牌有效期（秒），发给前端用。
func AccessTTL() int { return int(accessTTL.Seconds()) }

// HashPassword 生成 bcrypt 哈希（默认 cost=10，约 50-80ms，正好也是枚举防护的基准耗时）。
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword 校验口令。哈希为空表示这个账号没有口令（纯通行密钥账号），直接判失败。
func CheckPassword(hashed, pw string) bool {
	if hashed == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(pw)) == nil
}

// dummyHash 给"用户不存在"分支准备的等价哈希。
// 不做这一步，攻击者靠响应快慢就能枚举出哪些用户名真实存在。
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("qsl-no-such-user-placeholder"), bcrypt.DefaultCost)

// BurnPasswordTime 跑一次和真实校验等价的 bcrypt，抹平快慢路径的时间差。
func BurnPasswordTime(pw string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(pw))
}

// Claims 自定义令牌声明。字段名保持 sub/exp/iat/iss，老令牌照样能读。
type Claims struct {
	AMR  []string `json:"amr,omitempty"`
	Ver  int      `json:"ver"`
	Type string   `json:"type,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken 同时签发访问令牌和刷新令牌，两者携带相同的认证方式与版本号。
func GenerateToken(username string, version int, amr []string) (string, string, error) {
	now := time.Now()
	access := Claims{AMR: amr, Ver: version, RegisteredClaims: jwt.RegisteredClaims{
		Subject:   username,
		Issuer:    tokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(accessTTL)),
	}}
	refresh := Claims{AMR: amr, Ver: version, Type: "refresh", RegisteredClaims: jwt.RegisteredClaims{
		Subject:   username,
		Issuer:    tokenIssuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(refreshTTL)),
	}}

	accessStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, access).SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}
	refreshStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refresh).SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}
	return accessStr, refreshStr, nil
}

// parseToken 校验签名并解出声明。算法固定 HS256，防算法混淆攻击。
func parseToken(raw string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非法签名算法")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// ParseRefreshToken 解出刷新令牌，是否还有效由调用方查库判断。
func ParseRefreshToken(raw string) (*Claims, error) {
	claims, err := parseToken(raw)
	if err != nil {
		return nil, err
	}
	if claims.Type != "refresh" {
		return nil, fmt.Errorf("不是刷新令牌")
	}
	return claims, nil
}

func bearerToken(header string) (string, bool) {
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return tok, tok != ""
}

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"code": "QSL-401-0001", "message": "认证失败"})
	c.Abort()
}

// AuthRequired 校验令牌，并把 username / amr 放进上下文。
//
// 除了验签名，还要查一次库：账号是否还在、是否被停用、令牌版本有没有被顶掉。
// 只验签名的话，删号、停用、改密都拦不住已经发出去的令牌。
// SQLite 主键点查是微秒级，这点开销换"改完立刻生效"，值得。
func AuthRequired(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			unauthorized(c)
			return
		}
		claims, err := parseToken(raw)
		if err != nil || claims.Type == "refresh" {
			unauthorized(c)
			return
		}

		var user model.User
		if err := db.Where("username = ?", claims.Subject).First(&user).Error; err != nil || !user.IsActive {
			unauthorized(c)
			return
		}
		if claims.Ver != user.TokenVersion {
			unauthorized(c)
			return
		}

		c.Set("username", user.Username)
		c.Set("amr", claims.AMR)
		c.Next()
	}
}

// HasSecondFactor 判断当前这次登录有没有经过第二因子。
func HasSecondFactor(c *gin.Context) bool {
	v, ok := c.Get("amr")
	if !ok {
		return false
	}
	list, _ := v.([]string)
	for _, m := range list {
		if m == "totp" || m == "passkey" {
			return true
		}
	}
	return false
}

// AccountHasSecondFactor 账号本身是否绑了任一第二因子。
func AccountHasSecondFactor(db *gorm.DB, username string) bool {
	var user model.User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return false
	}
	var n int64
	db.Model(&model.Passkey{}).Where("user_id = ?", user.ID).Count(&n)
	if n > 0 {
		return true
	}
	db.Model(&model.TotpSecret{}).Where("user_id = ? AND enabled = ?", user.ID, true).Count(&n)
	return n > 0
}

// RequireMFA 挂在全量导出、导入、SMTP 凭据这类"一旦拿到就全盘失守"的接口上。
//
// force=false（默认）：账号还没绑任何第二因子时放行——否则刚装好的系统连备份都导不出来。
// force=true（REQUIRE_MFA=1）：一律要求，管理员必须先去绑定二步验证才能动这些数据。
func RequireMFA(db *gorm.DB, force bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if HasSecondFactor(c) {
			c.Next()
			return
		}
		if !force && !AccountHasSecondFactor(db, c.GetString("username")) {
			c.Next()
			return
		}
		c.JSON(http.StatusForbidden, gin.H{"code": "QSL-403-0002", "message": "该操作需要二次验证"})
		c.Abort()
	}
}

// CORSMiddleware 仅允许本地开发端口跨域。生产同源部署用不到它。
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:8000",
			"http://127.0.0.1:8000",
		}

		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.AbortWithStatus(204)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}
		c.Next()
	}
}
