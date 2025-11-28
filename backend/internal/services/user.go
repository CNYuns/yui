package services

import (
	"errors"

	"xpanel/internal/database"
	"xpanel/internal/models"
	"xpanel/pkg/utils"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin operator viewer"`
	Nickname string `json:"nickname"`
}

type UpdateUserRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"omitempty,min=6"`
	Role     string `json:"role" binding:"omitempty,oneof=admin operator viewer"`
	Nickname string `json:"nickname"`
	Status   *int   `json:"status"`
}

// List 获取用户列表
func (s *UserService) List(page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	db := database.DB.Model(&models.User{})
	db.Count(&total)

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Get 获取单个用户
func (s *UserService) Get(id uint) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	return &user, nil
}

// Create 创建用户
func (s *UserService) Create(req *CreateUserRequest) (*models.User, error) {
	var count int64
	database.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		return nil, errors.New("邮箱已存在")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := models.User{
		Email:    req.Email,
		Password: hashedPassword,
		Role:     req.Role,
		Nickname: req.Nickname,
		Status:   1,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Update 更新用户
func (s *UserService) Update(id uint, req *UpdateUserRequest) (*models.User, error) {
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	updates := make(map[string]interface{})

	if req.Email != "" && req.Email != user.Email {
		var count int64
		database.DB.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, id).Count(&count)
		if count > 0 {
			return nil, errors.New("邮箱已存在")
		}
		updates["email"] = req.Email
	}

	if req.Password != "" {
		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			return nil, err
		}
		updates["password"] = hashedPassword
	}

	if req.Role != "" {
		updates["role"] = req.Role
	}

	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}

	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	database.DB.First(&user, id)
	return &user, nil
}

// Delete 删除用户
func (s *UserService) Delete(id uint) error {
	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return errors.New("用户不存在")
	}

	return database.DB.Delete(&user).Error
}
