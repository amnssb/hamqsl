package handler

import (
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"qsl-management/internal/middleware"
	"qsl-management/internal/model"
	"qsl-management/pkg/response"
)

type AuthHandler struct{ db *gorm.DB }

func NewAuthHandler(db *gorm.DB) *AuthHandler { return &AuthHandler{db: db} }

// writeAudit 写入审计日志（失败仅记服务端日志，不影响主流程）
func writeAudit(db *gorm.DB, action, operator, detail, ip string) {
	entry := model.AuditLog{
		Action:       action,
		ResourceType: "auth",
		Operator:     operator,
		Detail:       detail,
		IPAddress:    ip,
		CreatedAt:    cstNow(),
	}
	if err := db.Create(&entry).Error; err != nil {
		log.Printf("[audit] 写入失败 action=%s err=%v", action, err)
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct{ Username, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "参数错误")
		return
	}
	var user model.User
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		writeAudit(h.db, "LOGIN_FAILED", req.Username, "用户不存在", c.ClientIP())
		response.Unauthorized(c)
		return
	}
	if !middleware.CheckPassword(user.HashPassword, req.Password) {
		writeAudit(h.db, "LOGIN_FAILED", req.Username, "密码错误", c.ClientIP())
		response.Unauthorized(c)
		return
	}
	accessToken, refreshToken, err := middleware.GenerateToken(user.Username)
	if err != nil {
		response.Error(c, "生成令牌失败")
		return
	}
	writeAudit(h.db, "LOGIN_OK", user.Username, "登录成功", c.ClientIP())
	response.OK(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "bearer",
		"expires_in":    7200, // 2小时 = 7200秒
	})
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
	writeAudit(h.db, "CHANGE_PASSWORD", user.Username, "修改密码成功", c.ClientIP())
	response.OKMsg(c, "密码已修改，请重新登录")
}

// Refresh 使用刷新令牌换取新的令牌对（密码修改后签发的旧刷新令牌立即失效）
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.RefreshToken) == "" {
		response.Fail(c, 400, "缺少刷新令牌")
		return
	}
	access, refresh, err := middleware.RefreshToken(strings.TrimSpace(req.RefreshToken), func(username string, issuedAt int64) error {
		var user model.User
		if err := h.db.Where("username = ?", username).First(&user).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}
		// 令牌签发早于用户记录最后更新（改密会触发 Save/UpdatedAt）→ 旧刷新令牌作废
		if issuedAt > 0 && user.UpdatedAt.Unix() > issuedAt {
			return fmt.Errorf("密码已变更，请重新登录")
		}
		return nil
	})
	if err != nil {
		response.Fail(c, 401, err.Error())
		return
	}
	response.OK(c, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"token_type":    "bearer",
		"expires_in":    7200,
	})
}
