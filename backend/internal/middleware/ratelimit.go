package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // 每分钟最大请求数
	burst    int           // 突发请求上限
	cleanup  time.Duration // 清理间隔
}

type visitor struct {
	count    int
	lastSeen time.Time
}

// NewRateLimiter 创建新的速率限制器
func NewRateLimiter(rate, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
		cleanup:  3 * time.Minute,
	}
	go rl.cleanupVisitors()
	return rl
}

// cleanupVisitors 定期清理过期的访问者记录
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.cleanup {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// isAllowed 检查请求是否允许
func (rl *RateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
		return true
	}

	// 检查是否需要重置计数器（每分钟重置）
	if time.Since(v.lastSeen) > time.Minute {
		v.count = 1
		v.lastSeen = time.Now()
		return true
	}

	// 检查是否超过限制
	if v.count >= rl.rate {
		return false
	}

	v.count++
	v.lastSeen = time.Now()
	return true
}

// RateLimitMiddleware 创建速率限制中间件
func RateLimitMiddleware(rate, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(rate, burst)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.isAllowed(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    "QSL-429",
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// LoginRateLimitMiddleware 登录接口专用的严格速率限制
func LoginRateLimitMiddleware() gin.HandlerFunc {
	// 登录接口：每分钟最多5次尝试，突发最多3次
	return RateLimitMiddleware(5, 3)
}

// APIRateLimitMiddleware 普通API接口的速率限制
// 认证后的后台 API 为单管理员使用场景，放宽到 600 次/分钟：
// 正常交互远低于该值，同时仍能拦截脚本化滥用与意外死循环请求。
func APIRateLimitMiddleware() gin.HandlerFunc {
	// 后台API：每分钟最多600次请求
	return RateLimitMiddleware(600, 100)
}