package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret []byte

func SetSecret(secret string) {
	jwtSecret = []byte(secret)
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hashed, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(pw)) == nil
}

func GenerateToken(username string) (string, string, error) {
	// 访问令牌：有效期2小时
	accessClaims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(2 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"iss": "qsl-management",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	// 刷新令牌：有效期7天
	refreshClaims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
		"iss": "qsl-management",
		"type": "refresh",
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshString, err := refreshToken.SignedString(jwtSecret)
	if err != nil {
		return "", "", err
	}

	return accessString, refreshString, nil
}

// RefreshToken 校验并使用刷新令牌换取新令牌对。
// check 回调用于业务侧额外校验（如：令牌签发后密码是否已变更），返回错误则拒绝刷新。
func RefreshToken(refreshTokenString string, check func(username string, issuedAt int64) error) (string, string, error) {
	token, err := jwt.Parse(refreshTokenString, func(t *jwt.Token) (interface{}, error) {
		// 算法固定 HS256，防算法混淆攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非法签名算法")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", "", fmt.Errorf("无效的刷新令牌")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("无效的令牌声明")
	}

	// 检查令牌类型
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return "", "", fmt.Errorf("不是刷新令牌")
	}

	username, ok := claims["sub"].(string)
	if !ok || username == "" {
		return "", "", fmt.Errorf("无效的用户名")
	}

	iat := int64(0)
	if f, ok := claims["iat"].(float64); ok {
		iat = int64(f)
	}

	if check != nil {
		if err := check(username, iat); err != nil {
			return "", "", err
		}
	}

	// 生成新的令牌对
	return GenerateToken(username)
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "QSL-401-0001", "message": "认证失败"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			// 算法固定 HS256，防算法混淆攻击
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("非法签名算法")
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "QSL-401-0001", "message": "认证失败"})
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "QSL-401-0001", "message": "认证失败"})
			c.Abort()
			return
		}
		// 刷新令牌不得充当访问令牌访问业务接口
		if tt, _ := claims["type"].(string); tt == "refresh" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": "QSL-401-0002", "message": "令牌类型错误，请使用访问令牌"})
			c.Abort()
			return
		}
		c.Set("username", claims["sub"])
		c.Next()
	}
}

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
