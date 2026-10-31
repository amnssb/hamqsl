package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type DashboardHandler struct{ db *gorm.DB }

func NewDashboardHandler(db *gorm.DB) *DashboardHandler { return &DashboardHandler{db: db} }

// Summary 总览指标。
//
// 以前这里是 8 条独立的 COUNT(*)，每条都把 card_records 扫一遍。
// 现在卡片相关的 6 个指标合并成一条聚合：
//   SUM(CASE WHEN ...) 而不是 COUNT(*) FILTER (...)，
//   因为 FILTER 语法 SQLite 老版本不支持，SUM(CASE) 两边都认。
func (h *DashboardHandler) Summary(c *gin.Context) {
	var agg struct {
		Total          int64
		Sent           int64
		Received       int64
		PendingIssue   int64
		PendingSent    int64
		PendingReceive int64
	}

	h.db.Model(&model.CardRecord{}).Select(`
		COUNT(*) AS total,
		COALESCE(SUM(CASE WHEN card_sent THEN 1 ELSE 0 END), 0) AS sent,
		COALESCE(SUM(CASE WHEN card_received THEN 1 ELSE 0 END), 0) AS received,
		COALESCE(SUM(CASE WHEN NOT card_issued AND call_sign <> '' THEN 1 ELSE 0 END), 0) AS pending_issue,
		COALESCE(SUM(CASE WHEN card_issued AND NOT card_sent THEN 1 ELSE 0 END), 0) AS pending_sent,
		COALESCE(SUM(CASE WHEN card_sent AND NOT card_received THEN 1 ELSE 0 END), 0) AS pending_receive
	`).Scan(&agg)

	var totalQso, pendingExchange, totalExchange, totalActivities int64
	h.db.Model(&model.QsoRecord{}).Count(&totalQso)
	h.db.Model(&model.ExchangeRequest{}).Count(&totalExchange)
	h.db.Model(&model.ExchangeRequest{}).Where("review_status = ?", "PENDING").Count(&pendingExchange)
	h.db.Model(&model.OfflineActivity{}).Count(&totalActivities)

	response.OK(c, gin.H{
		"total_qso":                totalQso,
		"total_cards":              agg.Total,
		"total_sent":               agg.Sent,
		"total_received":           agg.Received,
		"pending_issue":            agg.PendingIssue,
		"pending_sent":             agg.PendingSent,
		"pending_receive":          agg.PendingReceive,
		"pending_exchange_review":  pendingExchange,
		// 下面两项前端 Dashboard.vue 一直在读，但后端以前从没返回过 → 页面恒显示 0
		"total_exchange_requests":  totalExchange,
		"total_offline_activities": totalActivities,
	})
}
