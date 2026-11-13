package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type StationHandler struct{ db *gorm.DB }

func NewStationHandler(db *gorm.DB) *StationHandler { return &StationHandler{db: db} }

func (h *StationHandler) GetProfile(c *gin.Context) {
	var p model.StationProfile
	if err := h.db.First(&p).Error; err != nil {
		// An empty profile is a valid first-run state for the settings page.
		response.OK(c, model.StationProfile{})
		return
	}
	response.OK(c, p)
}

func (h *StationHandler) CreateProfile(c *gin.Context) {
	var req model.StationProfile
	c.ShouldBindJSON(&req)
	var count int64
	h.db.Model(&model.StationProfile{}).Count(&count)
	if count > 0 {
		response.Fail(c, 422, "本台资料已存在")
		return
	}
	h.db.Create(&req)
	response.OK(c, req)
}

func (h *StationHandler) UpdateProfile(c *gin.Context) {
	var item model.StationProfile
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "不存在")
		return
	}
	if ok, err := applyPatch(c, h.db, &item, c.Param("id"), stationProfileEditableFields); !ok {
		if err != nil {
			response.Fail(c, 500, "保存失败："+err.Error())
			return
		}
		response.Fail(c, 400, "没有可更新的字段")
		return
	}
	response.OK(c, item)
}

func (h *StationHandler) ListCards(c *gin.Context) {
	var items []model.StationCard
	h.db.Order("sort_order").Find(&items)
	response.OK(c, items)
}

func (h *StationHandler) CreateCard(c *gin.Context) {
	var req model.StationCard
	c.ShouldBindJSON(&req)
	h.db.Create(&req)
	response.OK(c, req)
}

func (h *StationHandler) UpdateCard(c *gin.Context) {
	var item model.StationCard
	if err := h.db.First(&item, c.Param("id")).Error; err != nil {
		response.NotFound(c, "不存在")
		return
	}
	if ok, err := applyPatch(c, h.db, &item, c.Param("id"), stationCardEditableFields); !ok {
		if err != nil {
			response.Fail(c, 500, "保存失败："+err.Error())
			return
		}
		response.Fail(c, 400, "没有可更新的字段")
		return
	}
	response.OK(c, item)
}

func (h *StationHandler) ListEquipments(c *gin.Context) {
	var items []model.StationEquipment
	h.db.Find(&items)
	response.OK(c, items)
}

func (h *StationHandler) CreateEquipment(c *gin.Context) {
	var req model.StationEquipment
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	h.db.Create(&req)
	response.OK(c, req)
}
