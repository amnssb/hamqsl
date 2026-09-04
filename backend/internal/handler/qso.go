package handler

import (
	"fmt"
	"strconv"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type QsoHandler struct{ db *gorm.DB }

func NewQsoHandler(db *gorm.DB) *QsoHandler { return &QsoHandler{db: db} }

func (h *QsoHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 { page = 1 }
	if size < 1 { size = 20 }

	q := h.db.Model(&model.QsoRecord{})
	if cs := c.Query("call_sign"); cs != "" {
		q = q.Where("call_sign LIKE ?", "%"+cs+"%")
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

// nextQsoRecordCode 生成下一个 QSO-XXXXXX 编号；加载已有序列行再更新
// （Save 新建 struct 会走 INSERT 撞唯一索引静默失败）
func nextQsoRecordCode(db *gorm.DB) string {
	var setting model.SystemSetting
	if err := db.Where("key = ?", "qso_record_sequence").First(&setting).Error; err != nil {
		setting = model.SystemSetting{Key: "qso_record_sequence", Value: "0"}
	}
	seq, _ := strconv.ParseInt(setting.Value, 10, 64)
	seq++
	setting.Value = strconv.FormatInt(seq, 10)
	db.Save(&setting)
	return fmt.Sprintf("QSO-%06d", seq)
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
