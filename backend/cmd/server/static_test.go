package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"qsl-management/internal/middleware"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// 最小 JPEG。扩展名 .jpg 时响应应为 image/jpeg，而不是去嗅探成别的类型。
var testJPEG = []byte{0xff, 0xd8, 0xff, 0xd9}

const (
	spaMarker = "SPA-INDEX-MARKER"
	secretTxt = "SECRET-BYTES-NOT-AN-IMAGE"
)

type harness struct {
	r *gin.Engine
}

func newHarness(t *testing.T, spa bool) *harness {
	t.Helper()
	root := t.TempDir()
	uploads := filepath.Join(root, "uploads")
	if err := os.MkdirAll(filepath.Join(uploads, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	write := func(path string, body []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(uploads, "card.jpg"), testJPEG)
	write(filepath.Join(uploads, "nested", "card.jpg"), testJPEG)
	write(filepath.Join(uploads, "my..card.jpg"), testJPEG)
	secretPath := filepath.Join(root, "secret.jpg")
	write(secretPath, []byte(secretTxt))
	if err := os.Symlink(secretPath, filepath.Join(uploads, "leak.jpg")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside")
	write(filepath.Join(outside, "passwd.jpg"), []byte(secretTxt))
	if err := os.Symlink(outside, filepath.Join(uploads, "linked")); err != nil {
		t.Fatal(err)
	}

	web := filepath.Join(root, "web")
	if spa {
		write(filepath.Join(web, "index.html"), []byte("<!doctype html><html><body>"+spaMarker+"</body></html>\n"))
		write(filepath.Join(web, "assets", "app.js"), []byte("console.log(1)\n"))
		write(filepath.Join(web, "favicon.ico"), []byte("ico"))
	} else {
		web = filepath.Join(root, "missing-web")
	}

	r := gin.New()
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())
	if got := mountSite(r, uploads, web); got != spa {
		t.Fatalf("mountSite=%v，想要 %v", got, spa)
	}
	r.GET("/api/testing-only", func(c *gin.Context) {
		c.String(http.StatusOK, "api-ok")
	})
	return &harness{r: r}
}

func (h *harness) do(method, pth, origin string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, pth, nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	h.r.ServeHTTP(w, req)
	return w
}

func assertMiss(t *testing.T, method, pth string, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusNotFound {
		t.Fatalf("%s %s: 状态 %d，想要 404；body=%q ctype=%q cache=%q",
			method, pth, w.Code, w.Body.String(), w.Header().Get("Content-Type"), w.Header().Get("Cache-Control"))
	}
	ct := strings.ToLower(w.Header().Get("Content-Type"))
	if strings.Contains(ct, "text/html") {
		t.Fatalf("%s %s: Content-Type 不应是 HTML：%q body=%q", method, pth, ct, w.Body.String())
	}
	if !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("%s %s: Content-Type=%q，想要 text/plain", method, pth, ct)
	}
	body := w.Body.String()
	lower := strings.ToLower(body)
	if strings.Contains(body, spaMarker) || strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<pre>") {
		t.Fatalf("%s %s: 回退成了页面：%s", method, pth, body)
	}
	if strings.Contains(body, secretTxt) {
		t.Fatalf("%s %s: 读到了上传目录外的文件", method, pth)
	}
	cc := w.Header().Get("Cache-Control")
	if strings.Contains(cc, "immutable") || strings.Contains(cc, "31536000") {
		t.Fatalf("%s %s: 缺失文件不应长缓存：%q", method, pth, cc)
	}
	if cc != "no-store" {
		t.Fatalf("%s %s: Cache-Control=%q，想要 no-store", method, pth, cc)
	}
	if method == http.MethodHead {
		if body != "" {
			t.Fatalf("HEAD %s: 不应有正文 %q", pth, body)
		}
		return
	}
	if body != "文件不存在\n" {
		t.Fatalf("%s %s: 正文 %q", method, pth, body)
	}
	if w.Header().Get("Cross-Origin-Resource-Policy") != "cross-origin" {
		t.Fatalf("%s %s: 404 也应允许跨源读取，实际 %v", method, pth, w.Header())
	}
}

func TestExistingUploadIsImageAndEmbeddable(t *testing.T) {
	h := newHarness(t, true)
	origins := []string{"https://ba4slt.cn", "http://localhost:3000", ""}
	paths := []string{"/uploads/card.jpg", "/uploads/card.jpg?v=1", "/uploads/nested/card.jpg", "/uploads/my..card.jpg"}
	for _, origin := range origins {
		for _, pth := range paths {
			w := h.do(http.MethodGet, pth, origin)
			if w.Code != http.StatusOK {
				t.Fatalf("GET %s origin=%q: 状态 %d body=%q", pth, origin, w.Code, w.Body.String())
			}
			if w.Body.String() != string(testJPEG) {
				t.Fatalf("GET %s: 正文不是图片字节 %q", pth, w.Body.String())
			}
			ct := w.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, "image/") {
				t.Fatalf("GET %s: Content-Type=%q，想要 image/*", pth, ct)
			}
			if ct != "image/jpeg" {
				t.Fatalf("GET %s: Content-Type=%q，想要 image/jpeg", pth, ct)
			}
			if w.Header().Get("Cache-Control") != uploadImmutableCache {
				t.Fatalf("GET %s: Cache-Control=%q", pth, w.Header().Get("Cache-Control"))
			}
			if w.Header().Get("Cross-Origin-Resource-Policy") != "cross-origin" {
				t.Fatalf("GET %s: CORP=%q", pth, w.Header().Get("Cross-Origin-Resource-Policy"))
			}
			if w.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Fatalf("GET %s origin=%q: ACAO=%q", pth, origin, w.Header().Get("Access-Control-Allow-Origin"))
			}
			if w.Header().Get("Access-Control-Allow-Credentials") != "" {
				t.Fatalf("GET %s origin=%q: 公开图片不能带凭据头 %q", pth, origin, w.Header().Get("Access-Control-Allow-Credentials"))
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("GET %s: 安全头被拿掉了 %v", pth, w.Header())
			}
			if strings.Contains(w.Body.String(), spaMarker) {
				t.Fatal("图片响应里出现了前端入口")
			}
		}
	}

	head := h.do(http.MethodHead, "/uploads/card.jpg", "https://ba4slt.cn")
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD 应 200 且无正文：%d %q", head.Code, head.Body.String())
	}
	if !strings.HasPrefix(head.Header().Get("Content-Type"), "image/") {
		t.Fatalf("HEAD Content-Type=%q", head.Header().Get("Content-Type"))
	}
	if head.Header().Get("Cache-Control") != uploadImmutableCache {
		t.Fatalf("HEAD 也应带强缓存：%q", head.Header().Get("Cache-Control"))
	}
}

func TestMissingUploadDoesNotFallBackToSPA(t *testing.T) {
	h := newHarness(t, true)
	misses := []struct{ method, path string }{
		{http.MethodGet, "/uploads/46953a5c5e25352cc9f6af28.jpg"},
		{http.MethodGet, "/uploads/missing.jpg"},
		{http.MethodGet, "/uploads/missing.jpg?x=1"},
		{http.MethodGet, "/uploads"},
		{http.MethodGet, "/uploads/"},
		{http.MethodGet, "/uploads/nested"},
		{http.MethodGet, "/uploads/nested/"},
		{http.MethodGet, "/uploads/nested/missing.jpg"},
		{http.MethodGet, "/uploads/leak.jpg"},
		{http.MethodGet, "/uploads/linked/passwd.jpg"},
		{http.MethodGet, "/uploads/../../secret.jpg"},
		{http.MethodGet, "/uploads/%2e%2e/%2e%2e/secret.jpg"},
		{http.MethodGet, "/uploads/foo/../../secret.jpg"},
		{http.MethodGet, "/nested/../uploads/missing.jpg"},
		{http.MethodGet, "/uploads/../web/index.html"},
		{http.MethodHead, "/uploads/missing.jpg"},
		{http.MethodPost, "/uploads/card.jpg"},
		{http.MethodPost, "/uploads/missing.jpg"},
		{http.MethodPut, "/uploads/nested/missing.jpg"},
	}
	for _, tc := range misses {
		assertMiss(t, tc.method, tc.path, h.do(tc.method, tc.path, "https://ba4slt.cn"))
	}

	// 前端路由和带 hash 的资源照旧，不能被这次修改带坏。
	page := h.do(http.MethodGet, "/showcase/luotianyi", "https://ba4slt.cn")
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), spaMarker) {
		t.Fatalf("前端路由应回退 index.html：%d %s", page.Code, page.Body.String())
	}
	if page.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("index.html 仍应 no-cache：%q", page.Header().Get("Cache-Control"))
	}
	if page.Header().Get("Access-Control-Allow-Origin") == "*" {
		t.Fatal("不应把整站页面放开跨源")
	}

	asset := h.do(http.MethodGet, "/assets/app.js", "")
	if asset.Code != http.StatusOK || asset.Body.String() != "console.log(1)\n" {
		t.Fatalf("前端资源应直出：%d %q", asset.Code, asset.Body.String())
	}
	if asset.Header().Get("Cache-Control") != uploadImmutableCache {
		t.Fatalf("带 hash 的资源仍应长缓存：%q", asset.Header().Get("Cache-Control"))
	}

	icon := h.do(http.MethodGet, "/favicon.ico", "")
	if icon.Code != http.StatusOK || icon.Header().Get("Cache-Control") != "public, max-age=86400" {
		t.Fatalf("普通静态文件应缓存一天：%d %q", icon.Code, icon.Header().Get("Cache-Control"))
	}
}

func TestUploadsDoNotLoosenAPI(t *testing.T) {
	h := newHarness(t, true)

	api := h.do(http.MethodGet, "/api/testing-only", "https://ba4slt.cn")
	if api.Code != http.StatusOK || api.Body.String() != "api-ok" {
		t.Fatalf("API 应照常：%d %s", api.Code, api.Body.String())
	}
	if api.Header().Get("Access-Control-Allow-Origin") != "" || api.Header().Get("Cross-Origin-Resource-Policy") != "" {
		t.Fatalf("站外来源不应读到普通 API：%v", api.Header())
	}
	if api.Header().Get("Cross-Origin-Opener-Policy") != "same-origin" {
		t.Fatalf("全站 COOP 应还在：%v", api.Header())
	}

	missing := h.do(http.MethodGet, "/api/does-not-exist", "https://ba4slt.cn")
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), "接口不存在") {
		t.Fatalf("未注册 API 应保持 JSON 404：%d %s", missing.Code, missing.Body.String())
	}
	if strings.Contains(missing.Body.String(), spaMarker) {
		t.Fatal("API 404 不应是首页")
	}
	if missing.Header().Get("Access-Control-Allow-Origin") == "*" || missing.Header().Get("Cross-Origin-Resource-Policy") == "cross-origin" {
		t.Fatalf("API 404 不应套上公开嵌入头：%v", missing.Header())
	}
	ct := missing.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("API 404 Content-Type=%q", ct)
	}

	pre := h.do(http.MethodOptions, "/api/auth/login", "https://ba4slt.cn")
	if pre.Code != http.StatusForbidden {
		t.Fatalf("站外对登录接口的预检仍应拒绝，实际 %d", pre.Code)
	}

	// 开发来源的预检仍是 204，且带凭据；上传图的 GET 则必须把凭据头拿掉。
	devPre := h.do(http.MethodOptions, "/api/testing-only", "http://localhost:3000")
	if devPre.Code != http.StatusNoContent || devPre.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("本机预检应保持原样：%d %v", devPre.Code, devPre.Header())
	}
}

func TestMissingUploadWithoutFrontend(t *testing.T) {
	h := newHarness(t, false)
	w := h.do(http.MethodGet, "/uploads/46953a5c5e25352cc9f6af28.jpg", "")
	assertMiss(t, http.MethodGet, "/uploads/46953a5c5e25352cc9f6af28.jpg", w)

	page := h.do(http.MethodGet, "/showcase", "")
	if page.Code == http.StatusOK || strings.Contains(page.Body.String(), "<html") {
		t.Fatalf("没有 dist 时不应回退成页面：%d %s", page.Code, page.Body.String())
	}
}
