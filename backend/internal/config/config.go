package config

import (
	"crypto/rand"
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
}

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
		AdminPass:  getEnv("ADMIN_PASS", "admin123"),
	}
	if cfg.JWTSecret == "" {
		// 未显式提供密钥：自动生成并持久化（与数据库同目录，随数据卷持久）
		cfg.JWTSecret = loadOrCreateSecret(filepath.Dir(cfg.DBPath))
	}
	return cfg
}

// loadOrCreateSecret JWT 签名密钥：读取 data/.jwt_secret，不存在则生成 64 位随机
// hex 写入（0600）。密钥与数据库同生命周期：备份/迁移 data 目录即一并迁移。
// 注意：密钥文件丢失会导致所有已登录 token 失效，重新登录即可。
func loadOrCreateSecret(dir string) string {
	path := filepath.Join(dir, ".jwt_secret")
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

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
