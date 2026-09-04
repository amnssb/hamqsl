package handler

import (
	"strconv"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type AddressHandler struct{ db *gorm.DB }

func NewAddressHandler(db *gorm.DB) *AddressHandler { return &AddressHandler{db: db} }

func (h *AddressHandler) ListBook(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	q := h.db.Model(&model.AddressBookEntry{})
	if v := c.Query("call_sign"); v != "" { q = q.Where("call_sign LIKE ?", "%"+v+"%") }
	var total int64
	q.Count(&total)
	var items []model.AddressBookEntry
	q.Order("id DESC").Offset((page-1)*size).Limit(size).Find(&items)
	response.Page(c, items, total, page, size)
}

func (h *AddressHandler) CreateBook(c *gin.Context) {
	var req model.AddressBookEntry
	c.ShouldBindJSON(&req)
	h.db.Create(&req)
	response.OK(c, req)
}

func (h *AddressHandler) UpdateBook(c *gin.Context) {
	var item model.AddressBookEntry
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "地址不存在"); return
	}
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	h.db.Model(&item).Updates(req)
	response.OK(c, item)
}

func (h *AddressHandler) DeleteBook(c *gin.Context) {
	h.db.Delete(&model.AddressBookEntry{}, c.Param("id"))
	response.OKMsg(c, "删除成功")
}

func (h *AddressHandler) ListBureaus(c *gin.Context) {
	var items []model.BureauEntry
	h.db.Order("bureau_name").Find(&items)
	response.OK(c, items)
}

func (h *AddressHandler) CreateBureau(c *gin.Context) {
	var req model.BureauEntry
	c.ShouldBindJSON(&req)
	h.db.Create(&req)
	response.OK(c, req)
}

func (h *AddressHandler) UpdateBureau(c *gin.Context) {
	var item model.BureauEntry
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "卡片局不存在"); return
	}
	var req map[string]interface{}
	c.ShouldBindJSON(&req)
	h.db.Model(&item).Updates(req)
	response.OK(c, item)
}

func (h *AddressHandler) DeleteBureau(c *gin.Context) {
	h.db.Delete(&model.BureauEntry{}, c.Param("id"))
	response.OKMsg(c, "删除成功")
}
