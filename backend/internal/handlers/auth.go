package handlers

import (
	"y-ui/internal/database"
	"y-ui/internal/middleware"
	"y-ui/internal/models"
	"y-ui/internal/services"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		authService: services.NewAuthService(),
	}
}

// Login 登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	result, err := h.authService.Login(&req)
	if err != nil {
		response.Error(c, 1001, err.Error())
		return
	}

	response.Success(c, result)
}

// Logout 登出
func (h *AuthHandler) Logout(c *gin.Context) {
	// JWT 是无状态的，客户端删除 token 即可
	response.SuccessMsg(c, "登出成功")
}

// GetProfile 获取当前用户信息
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	userService := services.NewUserService()

	user, err := userService.Get(userID)
	if err != nil {
		response.Error(c, 1002, err.Error())
		return
	}

	response.Success(c, user)
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		response.Error(c, 1003, err.Error())
		return
	}

	response.SuccessMsg(c, "密码修改成功")
}

// InitAdmin 初始化管理员 (仅在无管理员时可用)
func (h *AuthHandler) InitAdmin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.authService.CreateInitAdmin(req.Email, req.Password); err != nil {
		response.Error(c, 1004, err.Error())
		return
	}

	response.SuccessMsg(c, "管理员创建成功")
}

// CheckInit 检查是否需要初始化
func (h *AuthHandler) CheckInit(c *gin.Context) {
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	response.Success(c, gin.H{
		"initialized": count > 0,
	})
}
