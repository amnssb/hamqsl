package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type QsoHandler struct{ db *gorm.DB }

func NewQsoHandler(db *gorm.DB) *QsoHandler { return &QsoHandler{db: db} }

func (h *QsoHandler) List(c *gin.Context) {
	page, size := pageParams(c)

	q := h.db.Model(&model.QsoRecord{})
	if cs := c.Query("call_sign"); cs != "" {
		// 前缀匹配（不加前置 %），才能吃到 call_sign 上的索引
		q = q.Where("call_sign LIKE ?", callSignPattern(cs))
	}
	if st := c.Query("scene_type"); st != "" {
		q = q.Where("scene_type = ?", st)
	}

	var total int64
	q.Count(&total)

	var items []model.QsoRecord
	q.Order("date DESC, time DESC").Offset((page - 1) * size).Limit(size).Find(&items)
	response.Page(c, items, total, page, size)
}

func (h *QsoHandler) Create(c *gin.Context) {
	var req model.QsoRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	req.RecordCode = nextQsoRecordCode(h.db)
	h.db.Create(&req)
	response.OK(c, req)
}

// nextQsoRecordCode 生成下一个 QSO-XXXXXX 编号（走 sequence.go 的统一发号器，
// 进程内互斥 + 显式 upsert，避免并发建卡时撞号）
func nextQsoRecordCode(db *gorm.DB) string {
	return nextSequence(db, "qso_record_sequence", "QSO-", 6, "")
}

func (h *QsoHandler) Get(c *gin.Context) {
	var item model.QsoRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "通联记录不存在")
		return
	}
	response.OK(c, item)
}

func (h *QsoHandler) Update(c *gin.Context) {
	var item model.QsoRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "通联记录不存在")
		return
	}
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	h.db.Model(&item).Updates(req)
	response.OK(c, item)
}

func (h *QsoHandler) Delete(c *gin.Context) {
	var item model.QsoRecord
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "通联记录不存在")
		return
	}
	if item.HasCard {
		response.Fail(c, 422, "已建卡的通联记录不能删除")
		return
	}
	h.db.Delete(&item)
	response.OKMsg(c, "删除成功")
}
