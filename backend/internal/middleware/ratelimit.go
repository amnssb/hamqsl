package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

/*
限速器：固定窗口计数。

窗口从某个键的第一次请求开始计时，到期整体清零。比"距上次请求超过 N 秒才清零"好理解，
也不会出现"一直被拒、窗口永远不重置"的怪状态。

调用示例：

	loginIPLimiter.Allow("203.0.113.7")   // true 放行 / false 拒绝
	loginUserLimiter.Reset("admin")       // 登录成功后清掉这个账号的计数
*/
type windowLimiter struct {
	limit     int
	window    time.Duration
	maxKeys   int
	mu        sync.Mutex
	entries   map[string]*windowEntry
	lastSweep time.Time
}

type windowEntry struct {
	count   int
	started time.Time
}

func newWindowLimiter(limit int, window time.Duration, maxKeys int) *windowLimiter {
	return &windowLimiter{
		limit:   limit,
		window:  window,
		maxKeys: maxKeys,
		entries: make(map[string]*windowEntry),
	}
}

// Allow 记一次并判断是否放行。
func (l *windowLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweepLocked(now) // 先清过期的，腾出位置

	e, ok := l.entries[key]
	if !ok {
		// 键数量封顶：伪造键（比如随手换 X-Forwarded-For）不能把内存撑爆。
		// 满了就拒绝新键，宁让别人稍后重试，也不无限扩张。
		if len(l.entries) >= l.maxKeys {
			return false
		}
		l.entries[key] = &windowEntry{count: 1, started: now} // 窗口从这一次开始算
		return true
	}
	if now.Sub(e.started) >= l.window { // 窗口到期，重新开始计数
		e.count = 1
		e.started = now
		return true
	}
	if e.count >= l.limit { // 配额用完
		return false
	}
	e.count++
	return true
}

// Reset 清掉一个键的计数。登录成功时调用，避免正常用户被自己的重试拖累。
func (l *windowLimiter) Reset(key string) {
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}

// sweepLocked 清过期键。只在表长到一半容量、且距上次清理超过 1 秒时才做，避免每次请求都遍历。
func (l *windowLimiter) sweepLocked(now time.Time) {
	if len(l.entries) < l.maxKeys/2 || now.Sub(l.lastSweep) < time.Second {
		return
	}
	l.lastSweep = now
	for k, e := range l.entries {
		if now.Sub(e.started) >= l.window {
			delete(l.entries, k)
		}
	}
}

// --- 三个桶 ---
//
// 登录要同时过 IP 桶和账号桶，任一超限就拒。
// 只按 IP 限速等于没限：攻击者换个 X-Forwarded-For 就换了身份。
// 账号桶与 IP 无关，伪造多少头都没用，这才是真正挡住定向爆破的那一半。

var (
	loginIPLimiter   = newWindowLimiter(5, time.Minute, 50000)   // 同一 IP：5 次/分钟
	loginUserLimiter = newWindowLimiter(5, 5*time.Minute, 10000) // 同一账号：5 次/5分钟
	totpLimiter      = newWindowLimiter(5, 5*time.Minute, 10000) // 验证码：6 位数字只有 100 万种，必须限
	apiLimiter       = newWindowLimiter(600, time.Minute, 50000) // 已登录后台接口：600 次/分钟
)

// AllowLoginAttempt 账号维度配额。用户名大小写不敏感，避免 Admin / admin 各算一份。
func AllowLoginAttempt(username string) bool {
	return loginUserLimiter.Allow(strings.ToLower(strings.TrimSpace(username)))
}

// ResetLoginAttempt 登录成功后清掉账号配额。
func ResetLoginAttempt(username string) {
	loginUserLimiter.Reset(strings.ToLower(strings.TrimSpace(username)))
}

// AllowTotpAttempt 二步验证码的尝试配额，按账号计。
func AllowTotpAttempt(username string) bool {
	return totpLimiter.Allow(strings.ToLower(strings.TrimSpace(username)))
}

func tooMany(c *gin.Context) {
	c.Header("Retry-After", "60")
	c.JSON(http.StatusTooManyRequests, gin.H{
		"code":    "QSL-429",
		"message": "请求过于频繁，请稍后再试",
	})
	c.Abort()
}

// LoginRateLimitMiddleware 登录相关接口的 IP 维度限速。
// 账号维度在 handler 里（要先解析出 username 才能计数）。
func LoginRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !loginIPLimiter.Allow(RealClientIP(c)) {
			tooMany(c)
			return
		}
		c.Next()
	}
}

// APIRateLimitMiddleware 认证后的后台接口限速。
// 单管理员使用场景，正常交互远低于 600 次/分钟，只拦脚本化滥用与死循环。
func APIRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !apiLimiter.Allow(RealClientIP(c)) {
			tooMany(c)
			return
		}
		c.Next()
	}
}
