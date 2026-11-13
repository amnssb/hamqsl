package plugins

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

func init() {
	// 60：排在五个主题后面。数字只决定插件页上的先后，不参与主题谁盖住谁。
	plugin.Add(60, func(db *gorm.DB) plugin.Plugin { return &StatsDaily{db: db} })
}

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
	response.OK(c, gin.H{"days": days, "items": aggregateCardDays(p.db, days, time.Now())})
}

// aggregateCardDays 统计最近 days 天（含今天，东八区）的建卡/已制卡/已寄出/已签收。
//
// 计数口径和以前的逐日 COUNT 一样：按 created_at 落入哪一天来分桶，
// 制卡/寄出/签收看的是这批卡现在的布尔位，不是状态变更发生在哪一天。
// 以前 14 天 × 4 个 COUNT = 56 次扫描；现在窗口内的行只读一次，再在内存里分桶。
func aggregateCardDays(db *gorm.DB, days int, now time.Time) []dayRow {
	now = now.In(cstZone)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, cstZone)
	rangeStart := todayStart.AddDate(0, 0, -(days - 1))
	rangeEnd := todayStart.AddDate(0, 0, 1)

	items := make([]dayRow, 0, days)
	index := make(map[string]int, days)
	for i := days - 1; i >= 0; i-- {
		start := todayStart.AddDate(0, 0, -i)
		index[start.Format("2006-01-02")] = len(items)
		items = append(items, dayRow{Day: start.Format("2006-01-02")})
	}

	var rows []struct {
		CreatedAt        time.Time
		CardIssued       bool
		CardSent         bool
		ReceiptConfirmed bool
	}
	db.Model(&model.CardRecord{}).
		Select("created_at, card_issued, card_sent, receipt_confirmed").
		Where("created_at >= ? AND created_at < ?", rangeStart, rangeEnd).
		Find(&rows)

	for _, r := range rows {
		day := r.CreatedAt.In(cstZone).Format("2006-01-02")
		i, ok := index[day]
		if !ok {
			continue // 落在窗口外（驱动解析时区有偏差时宁可丢掉，也不串到别的天）
		}
		items[i].Created++
		if r.CardIssued {
			items[i].Issued++
		}
		if r.CardSent {
			items[i].Sent++
		}
		if r.ReceiptConfirmed {
			items[i].Signed++
		}
	}
	return items
}
