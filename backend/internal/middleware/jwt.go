package middleware

import (
	"errors"
	"strings"
	"sync"
	"time"

	"y-ui/internal/config"
	"y-ui/internal/database"
	"y-ui/internal/models"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

const (
	ContextUserID = "user_id"
	ContextEmail  = "email"
	ContextRole   = "role"
)

// TokenBlacklist Token 黑名单（用于登出后立即失效）
type TokenBlacklist struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
}

var tokenBlacklist = &TokenBlacklist{
	tokens: make(map[string]time.Time),
}

// Add 添加 Token 到黑名单
func (tb *TokenBlacklist) Add(token string, expiresAt time.Time) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.tokens[token] = expiresAt

	// 清理过期的黑名单条目
	now := time.Now()
	for t, exp := range tb.tokens {
		if exp.Before(now) {
			delete(tb.tokens, t)
		}
	}
}

// IsBlacklisted 检查 Token 是否在黑名单中
func (tb *TokenBlacklist) IsBlacklisted(token string) bool {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	expiresAt, exists := tb.tokens[token]
	if !exists {
		return false
	}
	// 如果已过期，不算在黑名单中
	return expiresAt.After(time.Now())
}

// BlacklistToken 将 Token 加入黑名单
func BlacklistToken(token string, expiresAt time.Time) {
	tokenBlacklist.Add(token, expiresAt)
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID uint, email, role string) (string, error) {
	cfg := config.GlobalConfig
	if cfg.Auth.JWTSecret == "" {
		return "", errors.New("jwt secret not configured")
	}

	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.Auth.TokenTTLHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "y-ui",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Auth.JWTSecret))
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	cfg := config.GlobalConfig
	if cfg.Auth.JWTSecret == "" {
		return nil, errors.New("jwt secret not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// JWTAuth JWT 认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "未提供认证信息")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.Unauthorized(c, "认证格式错误")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 检查 Token 是否在黑名单中
		if tokenBlacklist.IsBlacklisted(tokenString) {
			response.Unauthorized(c, "Token已失效，请重新登录")
			c.Abort()
			return
		}

		claims, err := ParseToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "无效的Token")
			c.Abort()
			return
		}

		// 验证用户状态是否仍为活跃
		var user models.User
		if err := database.DB.Select("id, status").First(&user, claims.UserID).Error; err != nil {
			response.Unauthorized(c, "用户不存在")
			c.Abort()
			return
		}
		if user.Status != 1 {
			response.Unauthorized(c, "账号已被禁用")
			c.Abort()
			return
		}

		// 将用户信息和原始 Token 存入上下文
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRole, claims.Role)
		c.Set("token", tokenString)
		c.Set("token_expires_at", claims.ExpiresAt.Time)

		c.Next()
	}
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) uint {
	if id, exists := c.Get(ContextUserID); exists {
		return id.(uint)
	}
	return 0
}

// GetUserRole 从上下文获取用户角色
func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get(ContextRole); exists {
		return role.(string)
	}
	return ""
}
