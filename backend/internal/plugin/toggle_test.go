package plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"qsl-management/internal/model"
)

// togglingPlugin 记录 Init 被调用的次数；initErr 让 Init 第一次返回失败。
type togglingPlugin struct {
	name    string
	inits   int
	initErr error
}

func (p *togglingPlugin) Meta() Info { return Info{Name: p.name, Kind: "feature"} }
func (p *togglingPlugin) Init(*gorm.DB) error {
	p.inits++
	return p.initErr
}
func (p *togglingPlugin) Routes(*gin.RouterGroup) {}

// newToggleManager 建一个只迁移 system_settings 的内存库，注册插件并挂好管理路由。
func newToggleManager(t *testing.T, ps ...*togglingPlugin) (*Manager, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(gormsqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	for _, p := range ps {
		m.Register(p)
	}
	r := gin.New()
	m.Mount(r.Group("/api"))
	return m, r
}

func TestEnableRunsInitImmediately(t *testing.T) {
	p := &togglingPlugin{name: "lazy"}
	m, _ := newToggleManager(t, p)

	// 新插件没启用过，启动路径（InitEnabled）不会碰它
	if m.IsEnabled("lazy") {
		t.Fatal("未启用时不应为 true")
	}
	if err := m.SetEnabled("lazy", true); err != nil {
		t.Fatalf("启用失败: %v", err)
	}
	if p.inits != 1 {
		t.Fatalf("启用时应立刻跑一次 Init，实际 %d 次", p.inits)
	}
	if !m.IsEnabled("lazy") {
		t.Fatal("启用后应读回 true")
	}
}

func TestEnableInitFailureRollsBackSwitch(t *testing.T) {
	p := &togglingPlugin{name: "broken", initErr: errors.New("建表失败")}
	m, _ := newToggleManager(t, p)

	if err := m.SetEnabled("broken", true); err == nil {
		t.Fatal("Init 失败应把错误带回来")
	}
	if p.inits != 1 {
		t.Fatalf("Init 应被尝试过一次，实际 %d 次", p.inits)
	}
	if m.IsEnabled("broken") {
		t.Fatal("Init 失败后开关应已回退为禁用")
	}

	// 修好之后（不再失败）重新启用能成功
	p.initErr = nil
	if err := m.SetEnabled("broken", true); err != nil {
		t.Fatalf("重试启用失败: %v", err)
	}
	if !m.IsEnabled("broken") {
		t.Fatal("重试后应读回 true")
	}
}

func TestToggleAPIReportsFailure(t *testing.T) {
	p := &togglingPlugin{name: "api", initErr: errors.New("预热失败")}
	m, _ := newToggleManager(t, p)

	// Mount 的管理端点带 AuthRequired，这里要测的是启停本身，
	// 单独挂一组不带认证的路由打到同一个 handler。
	free := gin.New()
	free.POST("/plugins/:name/enable", m.handleToggle(true))
	free.POST("/plugins/:name/disable", m.handleToggle(false))

	w := httptest.NewRecorder()
	free.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/plugins/api/enable", nil))
	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "预热失败") {
		t.Fatalf("Init 失败应 500 并带原因，实际 %d %s", w.Code, w.Body.String())
	}

	p.initErr = nil
	w = httptest.NewRecorder()
	free.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/plugins/api/enable", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "已启用") {
		t.Fatalf("修好后应启用成功，实际 %d %s", w.Code, w.Body.String())
	}

	// 未知插件仍 404
	w = httptest.NewRecorder()
	free.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/plugins/ghost/enable", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("未知插件应 404，实际 %d", w.Code)
	}
}
