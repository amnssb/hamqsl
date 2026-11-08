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
	// 通联换卡只要还没建卡的。不在这里滤，前端只能把已建卡的也拉下来再丢掉，
	// 最近 100 条都建过卡时，更早的未建卡记录根本出不来。
	if v := c.Query("has_card"); v == "true" || v == "false" {
		q = q.Where("has_card = ?", v == "true")
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
	// 白名单见 patch.go：record_code 与 has_card 不接受外部写入
	if ok, err := applyPatch(c, h.db, &item, c.Param("id"), qsoEditableFields); !ok {
		if err != nil {
			response.Fail(c, 500, "保存失败："+err.Error())
			return
		}
		response.Fail(c, 400, "没有可更新的字段")
		return
	}
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
