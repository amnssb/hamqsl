package handler

import (
	"sync"
	"time"

	"gorm.io/gorm"
	"qsl-management/internal/model"
)

// system_settings 的内存快照。
//
// system_settings 是一张只有几十行的小表，但读它的地方特别多：
// 公开页一次 SiteInfo 要读 3 个键、快递查询要读 2 个键、每个 /api/ext/* 请求
// 还要读插件开关……全都是"单行 SELECT"。SQLite 上这类查询很便宜，
// 但白白多出的往返在弱网/慢盘部署上会攒成肉眼可见的延迟。
//
// 这里用 5 秒 TTL 的快照：绝大多数请求直接命中内存，
// 后台改完设置后调用 invalidateSettingsCache() 立刻生效（不用等 TTL）。
const settingsCacheTTL = 5 * time.Second

type settingsSnapshotCache struct {
	mu   sync.RWMutex
	at   time.Time
	data map[string]string
}

var settingsCache settingsSnapshotCache

func (c *settingsSnapshotCache) load(db *gorm.DB) map[string]string {
	var rows []model.SystemSetting
	// 只取 key/value 两列，避免把 description 之类的长文本也读上来
	if err := db.Model(&model.SystemSetting{}).Select("key, value").Find(&rows).Error; err != nil {
		// 读失败就返回空快照但不缓存失败结果，下次请求再试
		return map[string]string{}
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		m[r.Key] = r.Value
	}
	return m
}

func settingsSnapshot(db *gorm.DB) map[string]string {
	settingsCache.mu.RLock()
	if settingsCache.data != nil && time.Since(settingsCache.at) < settingsCacheTTL {
		m := settingsCache.data
		settingsCache.mu.RUnlock()
		return m
	}
	settingsCache.mu.RUnlock()

	m := settingsCache.load(db)

	settingsCache.mu.Lock()
	settingsCache.data = m
	settingsCache.at = time.Now()
	settingsCache.mu.Unlock()
	return m
}

// invalidateSettingsCache 让下一次读取重新查库。
// 所有写 system_settings 的地方（保存设置、插件开关、序列号自增）都会顺带触发，
// 所以后台改完立刻可见。
func invalidateSettingsCache() {
	settingsCache.mu.Lock()
	settingsCache.data = nil
	settingsCache.at = time.Time{}
	settingsCache.mu.Unlock()
}
