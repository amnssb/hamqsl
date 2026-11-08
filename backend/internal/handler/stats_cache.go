package handler

import (
	"sync"
	"time"

	"gorm.io/gorm"
	"qsl-management/internal/model"
)

/*
公开统计的内存缓存。

GET /api/public/stats 是首页那一块「LIVE 数据面板」，三条 COUNT(*) 全是全表扫：
card_records 两次、exchange_requests 一次。首页是最容易被反复刷的页面，
而这几个数字本来就不需要秒级新鲜度——晚 60 秒没人看得出来。

和 settings_cache.go 同一个思路：5 秒 / 60 秒 TTL 的快照 + 写路径主动失效。
这里用 60 秒，因为数据量只会随卡片数线性增长，而展示要求是"差不多实时"。
*/
const publicStatsTTL = 60 * time.Second

type publicStatsData struct {
	CardsSent       int64
	CardsSigned     int64
	PendingRequests int64
}

type publicStatsCache struct {
	mu  sync.RWMutex
	at  time.Time
	val publicStatsData
	ok  bool
}

var statsCache publicStatsCache

// publicStats 读缓存，过期则重新聚合。
func publicStats(db *gorm.DB) publicStatsData {
	statsCache.mu.RLock()
	if statsCache.ok && time.Since(statsCache.at) < publicStatsTTL {
		v := statsCache.val
		statsCache.mu.RUnlock()
		return v
	}
	statsCache.mu.RUnlock()

	// 过期后只放一个请求去扫表，其余请求等这一次的结果。
	statsCache.mu.Lock()
	defer statsCache.mu.Unlock()
	if statsCache.ok && time.Since(statsCache.at) < publicStatsTTL {
		return statsCache.val
	}

	var v publicStatsData

	// 卡片侧两次 COUNT 合成一次聚合，和 dashboard.go 用的是同一套 SUM(CASE WHEN)，
	// 因为 FILTER 语法 SQLite 老版本不支持。
	var agg struct {
		Sent   int64
		Signed int64
	}
	if err := db.Model(&model.CardRecord{}).Select(`
		COALESCE(SUM(CASE WHEN card_sent THEN 1 ELSE 0 END), 0) AS sent,
		COALESCE(SUM(CASE WHEN receipt_confirmed THEN 1 ELSE 0 END), 0) AS signed
	`).Scan(&agg).Error; err != nil {
		if statsCache.ok {
			return statsCache.val // 这次没扫成就继续用上一份，不把首页数字刷成 0
		}
		return v
	}
	v.CardsSent = agg.Sent
	v.CardsSigned = agg.Signed

	if err := db.Model(&model.ExchangeRequest{}).Where("review_status = ?", "PENDING").Count(&v.PendingRequests).Error; err != nil {
		if statsCache.ok {
			return statsCache.val
		}
		return v
	}

	statsCache.val = v
	statsCache.at = time.Now()
	statsCache.ok = true
	return v
}

// invalidatePublicStatsCache 让下一次读取重新聚合。
//
// 在会改变这几个数字的写路径上调用（建卡/发信/签收/收卡/提交换卡申请）。
// 不调也不会错，只是数字最多晚 60 秒；调了则后台一操作，首页立刻对得上。
func invalidatePublicStatsCache() {
	statsCache.mu.Lock()
	statsCache.ok = false
	statsCache.at = time.Time{}
	statsCache.mu.Unlock()
}
