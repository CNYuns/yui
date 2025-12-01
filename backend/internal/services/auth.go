package services

import (
	"errors"
	"time"

	"y-ui/internal/config"
	"y-ui/internal/database"
	"y-ui/internal/middleware"
	"y-ui/internal/models"
	"y-ui/pkg/utils"
)

type AuthService struct{}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      *models.User `json:"user"`
}

func NewAuthService() *AuthService {
	return &AuthService{}
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	if user.Status != 1 {
		return nil, errors.New("账号已被禁用")
	}

	if !utils.CheckPassword(req.Password, user.Password) {
		return nil, errors.New("密码错误")
	}

	// 生成 Token
	token, err := middleware.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, errors.New("生成Token失败")
	}

	// 更新最后登录时间
	now := time.Now()
	database.DB.Model(&user).Update("last_login", now)

	return &LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(config.GlobalConfig.Auth.TokenTTLHours) * time.Hour),
		User:      &user,
	}, nil
}

// CreateInitAdmin 创建初始管理员
func (s *AuthService) CreateInitAdmin(username, password string) error {
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return errors.New("管理员已存在")
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	user := models.User{
		Username: username,
		Password: hashedPassword,
		Role:     middleware.RoleAdmin,
		Nickname: username,
		Status:   1,
	}

	return database.DB.Create(&user).Error
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	if !utils.CheckPassword(oldPassword, user.Password) {
		return errors.New("原密码错误")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return database.DB.Model(&user).Update("password", hashedPassword).Error
}
