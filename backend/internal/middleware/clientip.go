package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

/*
真实客户端地址。

规矩只有一条：**永远不读 X-Forwarded-For 的最左值。**
那个位置是客户端随手写的，谁都能填。gin 默认信任所有代理（TrustedProxies 为 0.0.0.0/0），
c.ClientIP() 返回的正是最左值，所以限速键、审计日志都不能用它。

这里改为：只有当对端是本机/内网反代（1Panel、nginx、docker 网桥）时，
才采纳反代用 proxy_set_header 覆盖写入的 X-Real-Client-IP；
直连过来的请求一律用对端地址，客户端塞什么头都不认。

调用示例：

	ip := middleware.RealClientIP(c)   // 例："203.0.113.7"
	loginIPLimiter.Allow(ip)
*/
func RealClientIP(c *gin.Context) string {
	peer := peerIP(c.Request)
	if !isLocalProxy(peer) {
		return peer // 直连：忽略一切客户端头
	}
	if v := strings.TrimSpace(c.GetHeader("X-Real-Client-IP")); net.ParseIP(v) != nil {
		return v
	}
	return peer
}

// peerIP 取 TCP 对端地址，去掉端口。
func peerIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// isLocalProxy 判断对端是不是本机或内网反代。
// 内网段也算，因为 docker 容器里的应用看到的反代地址通常是 172.17.0.1 这类网桥地址。
func isLocalProxy(ip string) bool {
	p := net.ParseIP(ip)
	if p == nil {
		return false
	}
	return p.IsLoopback() || p.IsPrivate()
}
