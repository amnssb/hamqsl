package plugin

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type namedPlugin struct{ name string }

func (p namedPlugin) Meta() Info            { return Info{Name: p.name, Kind: "feature"} }
func (namedPlugin) Init(*gorm.DB) error     { return nil }
func (namedPlugin) Routes(*gin.RouterGroup) {}

func TestInstallSortsByOrder(t *testing.T) {
	prev := candidates
	candidates = nil
	t.Cleanup(func() { candidates = prev }) // 别把测试用的插件留在进程里

	Add(30, func(*gorm.DB) Plugin { return namedPlugin{name: "c"} })
	Add(10, func(*gorm.DB) Plugin { return namedPlugin{name: "a"} })
	Add(20, func(*gorm.DB) Plugin { return namedPlugin{name: "b"} })

	m := NewManager(nil)
	m.Install(nil)
	if got := strings.Join(m.order, ","); got != "a,b,c" {
		t.Fatalf("应按数字从小到大装上，实际 %s", got)
	}
}
