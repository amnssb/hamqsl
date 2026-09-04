package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type AuthHandler struct{ db *gorm.DB }

func NewAuthHandler(db *gorm.DB) *AuthHandler { return &AuthHandler{db: db} }

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct{ Username, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	var user model.User
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		response.Unauthorized(c)
		return
	}
	if !middleware.CheckPassword(user.HashPassword, req.Password) {
		response.Unauthorized(c)
		return
	}
	token, err := middleware.GenerateToken(user.Username)
	if err != nil {
		response.Error(c, "生成令牌失败")
		return
	}
	response.OK(c, gin.H{"access_token": token, "token_type": "bearer"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	username, _ := c.Get("username")
	var user model.User
	if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
		response.Unauthorized(c)
		return
	}
	response.OK(c, gin.H{"id": user.ID, "username": user.Username, "display_name": user.DisplayName, "is_admin": user.IsAdmin})
}

// ChangePassword 当前登录用户修改自己的密码（需原密码验证）
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	if strings.TrimSpace(req.OldPassword) == "" {
		response.Fail(c, 400, "请输入原密码")
		return
	}
	if len(req.NewPassword) < 6 {
		response.Fail(c, 400, "新密码至少 6 位")
		return
	}
	var user model.User
	if err := h.db.Where("username = ?", c.GetString("username")).First(&user).Error; err != nil {
		response.Unauthorized(c)
		return
	}
	if !middleware.CheckPassword(user.HashPassword, req.OldPassword) {
		response.Fail(c, 400, "原密码错误")
		return
	}
	hash, err := middleware.HashPassword(req.NewPassword)
	if err != nil {
		response.Error(c, "密码加密失败")
		return
	}
	user.HashPassword = hash
	if err := h.db.Save(&user).Error; err != nil {
		response.Error(c, "保存失败，请稍后重试")
		return
	}
	response.OKMsg(c, "密码已修改，请重新登录")
}
