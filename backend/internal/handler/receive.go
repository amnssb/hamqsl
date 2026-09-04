package handler

import (
	"fmt"
	"strconv"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type ReceiveHandler struct{ db *gorm.DB }

func NewReceiveHandler(db *gorm.DB) *ReceiveHandler { return &ReceiveHandler{db: db} }

func (h *ReceiveHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	q := h.db.Model(&model.ReceiveRecord{})
	if v := c.Query("call_sign"); v != "" { q = q.Where("call_sign LIKE ?", "%"+v+"%") }
	var total int64
	q.Count(&total)
	var items []model.ReceiveRecord
	q.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&items)
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

// nextReceiveCode 生成收卡记录编号 R%04d-YYYYMMDD：更新 receive_record_sequence 序列行
func nextReceiveCode(db *gorm.DB, receivedDate string) string {
	// 更新已加载的序列行（Save 新建 struct 会走 INSERT 撞唯一索引静默失败）
	var setting model.SystemSetting
	if err := db.Where("key = ?", "receive_record_sequence").First(&setting).Error; err != nil {
		setting = model.SystemSetting{Key: "receive_record_sequence", Value: "0"}
	}
	seq, _ := strconv.Atoi(setting.Value)
	seq++
	setting.Value = strconv.Itoa(seq)
	db.Save(&setting)
	return fmt.Sprintf("R%04d-%s", seq, receivedDate[0:4]+receivedDate[5:7]+receivedDate[8:10])
}
