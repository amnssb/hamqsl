package statsdaily

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"

	"qsl-management/internal/plugin"
)

// StatsDaily 每日统计功能插件：向外提供 /api/ext/stats_daily/summary，
// 返回最近 N 天的卡片建卡/发卡/寄出/签收计数（东八区按天聚合）。
type StatsDaily struct{ db *gorm.DB }

func New(db *gorm.DB) *StatsDaily { return &StatsDaily{db: db} }

func (p *StatsDaily) Meta() plugin.Info {
	return plugin.Info{
		Name:        "stats_daily",
		Title:       "每日统计",
		Version:     "1.0.0",
		Description: "按天聚合卡片流转数据（建卡/发卡/寄出/签收），在插件页查看最近 14 天趋势。",
		Kind:        "feature",
	}
}

// Init 无需建表：直接基于既有 card_records 查询
func (p *StatsDaily) Init(db *gorm.DB) error { return nil }

func (p *StatsDaily) Routes(rg *gin.RouterGroup) {
	rg.GET("/summary", p.summary)
}

type dayRow struct {
	Day     string `json:"day"`
	Created int64  `json:"created"`
	Issued  int64  `json:"issued"`
	Sent    int64  `json:"sent"`
	Signed  int64  `json:"signed"`
}

// cstZone 东八区（与 handler.mail.go 的入库时间口径一致）
var cstZone = time.FixedZone("UTC+8", 8*3600)

func (p *StatsDaily) summary(c *gin.Context) {
	days := 14
	if d, err := strconv.Atoi(c.DefaultQuery("days", "14")); err == nil && d >= 1 && d <= 90 {
		days = d
	}

	now := time.Now().In(cstZone)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cstZone)

	items := make([]dayRow, 0, days)
	for i := days - 1; i >= 0; i-- {
		start := todayStart.AddDate(0, 0, -i)
		end := start.AddDate(0, 0, 1)
		items = append(items, dayRow{
			Day:     start.Format("2006-01-02"),
			Created: countRange(p.db, start, end, ""),
			Issued:  countRange(p.db, start, end, "card_issued = ?", true),
			Sent:    countRange(p.db, start, end, "card_sent = ?", true),
			Signed:  countRange(p.db, start, end, "receipt_confirmed = ?", true),
		})
	}
	response.OK(c, gin.H{"days": days, "items": items})
}

func countRange(db *gorm.DB, start, end time.Time, extra string, arg ...interface{}) int64 {
	q := db.Model(&model.CardRecord{}).Where("created_at >= ? AND created_at < ?", start, end)
	if extra != "" {
		q = q.Where(extra, arg...)
	}
	var n int64
	q.Count(&n)
	return n
}
