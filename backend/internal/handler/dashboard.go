package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type DashboardHandler struct{ db *gorm.DB }

func NewDashboardHandler(db *gorm.DB) *DashboardHandler { return &DashboardHandler{db: db} }

func (h *DashboardHandler) Summary(c *gin.Context) {
	var totalQso, totalCards, totalSent, totalReceived, pendingIssue, pendingSent, pendingReceive, pendingExchange int64
	h.db.Model(&model.QsoRecord{}).Count(&totalQso)
	h.db.Model(&model.CardRecord{}).Where("call_sign != ''").Count(&totalCards)
	h.db.Model(&model.CardRecord{}).Where("card_sent = true").Count(&totalSent)
	h.db.Model(&model.CardRecord{}).Where("card_received = true").Count(&totalReceived)
	h.db.Model(&model.CardRecord{}).Where("card_issued = false AND call_sign != ''").Count(&pendingIssue)
	h.db.Model(&model.CardRecord{}).Where("card_issued = true AND card_sent = false").Count(&pendingSent)
	h.db.Model(&model.CardRecord{}).Where("card_sent = true AND card_received = false").Count(&pendingReceive)
	h.db.Model(&model.ExchangeRequest{}).Where("review_status = 'PENDING'").Count(&pendingExchange)

	response.OK(c, gin.H{
		"total_qso": totalQso, "total_cards": totalCards,
		"total_sent": totalSent, "total_received": totalReceived,
		"pending_issue": pendingIssue, "pending_sent": pendingSent,
		"pending_receive": pendingReceive, "pending_exchange_review": pendingExchange,
	})
}
