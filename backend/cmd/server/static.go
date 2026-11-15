package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"qsl-management/internal/middleware"
)

// 命中的上传文件名是内容寻址的（随机 hex + 扩展名），可以长缓存。
// 缺失文件不能带这个头：卡片图补回磁盘之前，浏览器会把 404 记一年。
const uploadImmutableCache = "public, max-age=31536000, immutable"

// mountSite 挂上 /uploads，并在 webDir 存在时托管 Vue。
// 返回值表示是否在托管前端，根路径要据此决定返回页面还是 API 说明。
func mountSite(r *gin.Engine, uploadsDir, webDir string) bool {
	// 不用 r.Static。gin v1.10 的 createStaticHandler 在打开文件失败时，
	// 会把 c.handlers 换成 NoRoute 并把 index 拨回 -1（routergroup.go）。
	// Next 接着跑 SPA 回退，缺失的 /uploads/* 就变成 200 的 index.html。
	// 博客用 <img> 引卡片图时拿到的是 HTML，图就坏了。
	//
	// GET/HEAD 才出文件。其它方法也要占住这条路由，否则 POST 之类仍会掉进 NoRoute。
	h := serveUploads(prepareUploadsRoot(uploadsDir))
	r.Any("/uploads", h)
	r.Any("/uploads/*filepath", h)

	if _, err := os.Stat(webDir); err != nil {
		return false
	}
	r.NoRoute(spaFallback(webDir))
	log.Printf("前端静态目录: %s（单进程托管）", webDir)
	return true
}

func prepareUploadsRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return filepath.Clean(dir)
	}
	abs = filepath.Clean(abs)
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}

func serveUploads(root string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			uploadNotFound(c)
			return
		}
		full, ok := resolveUploadFile(root, c.Request.URL.Path)
		if !ok {
			uploadNotFound(c)
			return
		}
		f, err := os.Open(full)
		if err != nil {
			uploadNotFound(c)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil || !st.Mode().IsRegular() {
			uploadNotFound(c)
			return
		}
		// 卡片图要给 https://ba4slt.cn 用 <img> 跨源引用。
		// 全站 CORS 只放行本机开发端口，不能为了图片把 API 改成 *。
		// 这里只给这一条响应套和对外展示一样的公开嵌入头。
		// 全局的 COOP / CSP / X-Frame-Options 继续留着，它们不阻止跨源 <img>。
		middleware.AllowPublicEmbed(c)
		c.Header("Cache-Control", uploadImmutableCache)
		// ServeContent 按扩展名给 MIME，且不会因为 URL 以 index.html 结尾就改去回退页面。
		http.ServeContent(c.Writer, c.Request, st.Name(), st.ModTime(), f)
	}
}

func uploadNotFound(c *gin.Context) {
	middleware.AllowPublicEmbed(c)
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "text/plain; charset=utf-8")
	if c.Request.Method == http.MethodHead {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Data(http.StatusNotFound, "text/plain; charset=utf-8", []byte("文件不存在\n"))
	c.Abort()
}

// isUploadRequest 判断这个 URL 算不算 /uploads 下的请求。
// 原始路径以 /uploads 开头的算；path.Clean 之后仍落在 /uploads 里的也算
// （例如 /foo/../uploads/a.jpg）。这两种都不能回退成首页。
func isUploadRequest(urlPath string) bool {
	if urlPath == "/uploads" || strings.HasPrefix(urlPath, "/uploads/") {
		return true
	}
	if strings.ContainsRune(urlPath, 0) || strings.Contains(urlPath, `\`) {
		return false
	}
	cleaned := path.Clean(urlPath)
	return cleaned == "/uploads" || strings.HasPrefix(cleaned, "/uploads/")
}

// resolveUploadFile 只在 root 里面找到一个普通文件时返回它的真实路径。
// 目录、不存在、.. 逃逸、以及指向目录外的符号链接都返回 false。
func resolveUploadFile(root, urlPath string) (string, bool) {
	if strings.ContainsRune(urlPath, 0) || strings.Contains(urlPath, `\`) {
		return "", false
	}
	cleaned := path.Clean(urlPath)
	if !strings.HasPrefix(cleaned, "/uploads/") {
		return "", false
	}
	rel := strings.TrimPrefix(cleaned, "/uploads/")
	if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	candidate := filepath.Join(root, filepath.FromSlash(rel))
	if !withinDir(root, candidate) {
		return "", false
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", false
	}
	if !withinDir(root, resolved) {
		return "", false
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return resolved, true
}

func withinDir(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func spaFallback(webDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		// 上传路由没接住的方法、以及清洗后仍落在 /uploads 下的路径，
		// 都从这里挡掉。不能让它们变成 index.html。
		if isUploadRequest(reqPath) {
			uploadNotFound(c)
			return
		}
		// 命中的真实文件直接返回；/api 未注册路由保持 JSON 404；其余回退 index.html（history 路由）
		p := filepath.Join(webDir, filepath.Clean("/"+reqPath))
		if st, serr := os.Stat(p); serr == nil && !st.IsDir() {
			// /assets/ 下的文件名带内容 hash → 内容不可变，可以长缓存；
			// 其余静态文件（favicon 等）给一天。
			if strings.HasPrefix(reqPath, "/assets/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				c.Header("Cache-Control", "public, max-age=86400")
			}
			c.File(p)
			return
		}
		if len(reqPath) >= 5 && reqPath[:5] == "/api/" {
			c.JSON(404, gin.H{"code": "QSL-404", "message": "接口不存在"})
			return
		}
		// index.html 绝不能被缓存，否则发版后用户一直拿到旧入口
		c.Header("Cache-Control", "no-cache")
		c.File(filepath.Join(webDir, "index.html"))
	}
}
