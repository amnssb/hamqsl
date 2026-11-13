package plugin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qsl-management/internal/model"
)

type closedPlugin struct{}

func (closedPlugin) Meta() Info          { return Info{Name: "closed", Kind: "feature"} }
func (closedPlugin) Init(*gorm.DB) error { return nil }
func (closedPlugin) Routes(rg *gin.RouterGroup) {
	rg.GET("/ping", func(c *gin.Context) { c.Status(http.StatusNoContent) })
}

type openPlugin struct{}

func (openPlugin) Meta() Info              { return Info{Name: "open", Kind: "feature"} }
func (openPlugin) Init(*gorm.DB) error     { return nil }
func (openPlugin) Routes(*gin.RouterGroup) {}
func (openPlugin) OpenRoutes(rg *gin.RouterGroup) {
	rg.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
}

func TestMountOpenRequiresEnableAndNoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	m := NewManager(db)
	m.Register(closedPlugin{})
	m.Register(openPlugin{})
	r := gin.New()
	m.MountOpen(r.Group("/api/public"))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/ext/open/ping", nil))
	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), "QSL-PLUGIN-OFF") {
		t.Fatalf("未启用时应 404，实际 %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/ext/closed/ping", nil))
	if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "pong") {
		t.Fatalf("没实现 OpenReader 的插件不该出现在公开组，实际 %d %s", w.Code, w.Body.String())
	}

	m.SetEnabled("open", true)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/public/ext/open/ping", nil))
	if w.Code != http.StatusOK || w.Body.String() != "pong" {
		t.Fatalf("启用后无需登录应返回 pong，实际 %d %s", w.Code, w.Body.String())
	}
}
