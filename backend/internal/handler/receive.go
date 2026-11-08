package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type ReceiveHandler struct{ db *gorm.DB }

func NewReceiveHandler(db *gorm.DB) *ReceiveHandler { return &ReceiveHandler{db: db} }

func (h *ReceiveHandler) List(c *gin.Context) {
	page, size := pageParams(c)
	q := h.db.Model(&model.ReceiveRecord{})
	if v := c.Query("call_sign"); v != "" {
		q = q.Where("call_sign LIKE ?", callSignPattern(v))
	}
	var total int64
	q.Count(&total)
	var items []model.ReceiveRecord
	q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&items)
	response.Page(c, items, total, page, size)
}

func (h *ReceiveHandler) Create(c *gin.Context) {
	var req struct {
		CallSign       string `json:"call_sign"`
		CardType       string `json:"card_type"`
		BusinessType   string `json:"business_type"`
		ReceivedDate   string `json:"received_date"`
		OutboundCardID *uint  `json:"outbound_card_id"`
		Remarks        string `json:"remarks"`
	}
	c.ShouldBindJSON(&req)

	code := nextReceiveCode(h.db, req.ReceivedDate)

	record := model.ReceiveRecord{
		ReceiveCode: code, CallSign: req.CallSign, CardType: req.CardType,
		BusinessType: req.BusinessType, ReceivedDate: req.ReceivedDate,
		OutboundCardID: req.OutboundCardID, Remarks: req.Remarks, MatchStatus: "UNMATCHED",
	}
	if req.OutboundCardID != nil {
		var card model.CardRecord
		if err := h.db.First(&card, *req.OutboundCardID).Error; err == nil {
			card.CardReceived = true
			card.ReceivedAt = req.ReceivedDate
			card.ReceivedRecordCode = code
			h.db.Save(&card)
			record.MatchStatus = "MATCHED"
		}
	}
	h.db.Create(&record)
	response.OK(c, record)
}

// nextReceiveCode 生成收卡记录编号 R%04d-YYYYMMDD。
//
// 走 sequence.go 的统一发号器（进程内互斥 + 显式 upsert），
// 避免并发登记时两条记录拿到同一个编号。
func nextReceiveCode(db *gorm.DB, receivedDate string) string {
	// receivedDate 是用户输入；长度不足 10 时直接取当天，避免切片越界 panic
	suffix := ""
	if len(receivedDate) >= 10 && receivedDate[4] == '-' && receivedDate[7] == '-' {
		suffix = "-" + receivedDate[0:4] + receivedDate[5:7] + receivedDate[8:10]
	} else {
		today := cstNow().Format("2006-01-02")
		suffix = "-" + today[0:4] + today[5:7] + today[8:10]
	}
	return nextSequence(db, "receive_record_sequence", "R", 4, suffix)
}
