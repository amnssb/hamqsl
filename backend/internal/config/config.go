package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	DBDriver   string // sqlite or postgres
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPath     string // SQLite file path
	JWTSecret  string
	ServerPort string
	AdminUser  string
	AdminPass  string

	// WebAuthnRPID 通行密钥绑定的域（不含协议与端口，例：qsl.ba4slt.cn）。
	// 留空则整个通行密钥功能关闭——老部署升级上来不会因为少配环境变量起不来。
	WebAuthnRPID    string
	WebAuthnOrigins []string // 允许的完整来源，例：https://qsl.ba4slt.cn
	RequireMFA      bool     // 强制二步验证：没绑第二因子就不许登录
}

// mfaKey 二步验证密钥的加密主密钥，Load 时从磁盘读取或生成。
var mfaKey []byte

// MFAKey 返回 32 字节主密钥。丢了它，已存的 TOTP 密钥就解不开（需重新绑定）。
func MFAKey() []byte { return mfaKey }

func Load() *Config {
	cfg := &Config{
		DBDriver:   getEnv("DB_DRIVER", "sqlite"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "qsl"),
		DBPassword: getEnv("DB_PASSWORD", "qsl_secret_2024"),
		DBName:     getEnv("DB_NAME", "qsl_management"),
		DBPath:     getEnv("DB_PATH", "./data/qsl.db"),
		JWTSecret:  strings.TrimSpace(os.Getenv("JWT_SECRET")),
		ServerPort: getEnv("SERVER_PORT", "8000"),
		AdminUser:  getEnv("ADMIN_USER", "admin"),
		// 默认口令留空：由 main 在首次启动时随机生成并打印一次，
		// 不再用写死的 admin123 —— 那个值在公网上等于没有口令。
		AdminPass:       strings.TrimSpace(os.Getenv("ADMIN_PASS")),
		WebAuthnRPID:    strings.TrimSpace(os.Getenv("WEBAUTHN_RP_ID")),
		WebAuthnOrigins: splitList(os.Getenv("WEBAUTHN_ORIGINS")),
		RequireMFA:      strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_MFA")), "true"),
	}
	if cfg.JWTSecret == "" {
		// 未显式提供密钥：自动生成并持久化（与数据库同目录，随数据卷持久）
		cfg.JWTSecret = loadOrCreateFile(filepath.Dir(cfg.DBPath), ".jwt_secret")
	}
	mfaKey = deriveMFAKey(filepath.Dir(cfg.DBPath))
	return cfg
}

// deriveMFAKey 读取或生成 data/.mfa_key。
// 与 .jwt_secret 同目录、同生命周期：备份 data/ 目录就把密钥一起搬走，
// 不会出现"数据库搬过去了、二步验证全解不开"的情况。
func deriveMFAKey(dir string) []byte {
	text := strings.TrimSpace(loadOrCreateFile(dir, ".mfa_key"))
	b, err := hex.DecodeString(text)
	if err != nil || len(b) != 32 {
		// 文件被改坏：用它自身的摘要兜底，至少服务能起来。
		// 代价是已绑定的 TOTP 会解不开，需要重新绑定（有恢复码或 ADMIN_RESET 兜底）。
		sum := sha256.Sum256([]byte(text))
		return sum[:]
	}
	return b
}

// loadOrCreateFile 读取目录下的密钥文件；不存在则生成 64 位随机 hex 写入（0600）。
//
// 注意：密钥文件丢失会导致对应功能失效，重新生成即可。
func loadOrCreateFile(dir, name string) string {
	path := filepath.Join(dir, name)
	if b, err := os.ReadFile(path); err == nil {
		if s := strings.TrimSpace(string(b)); len(s) >= 32 {
			return s
		}
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// 极端兜底：crypto/rand 不可用时用时间熵（正常环境不会走到）
		return fmt.Sprintf("auto-%x", time.Now().UnixNano())
	}
	s := hex.EncodeToString(b)
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(path, []byte(s), 0o600)
	return s
}

// splitList 把逗号分隔的环境变量拆成列表，顺手去掉空项和尾斜杠。
func splitList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if v := strings.TrimRight(strings.TrimSpace(part), "/"); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
