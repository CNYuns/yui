package middleware

import (
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

// 角色权限定义
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
	RoleViewer   = "viewer"
)

// 权限级别
var rolePriority = map[string]int{
	RoleAdmin:    3,
	RoleOperator: 2,
	RoleViewer:   1,
}

// RequireRole 要求特定角色
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := GetUserRole(c)
		if userRole == "" {
			response.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "权限不足")
		c.Abort()
	}
}

// RequireMinRole 要求最低角色级别
func RequireMinRole(minRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := GetUserRole(c)
		if userRole == "" {
			response.Unauthorized(c, "未登录")
			c.Abort()
			return
		}

		userPriority, ok := rolePriority[userRole]
		if !ok {
			response.Forbidden(c, "无效的角色")
			c.Abort()
			return
		}

		minPriority, ok := rolePriority[minRole]
		if !ok {
			response.ServerError(c, "服务器配置错误")
			c.Abort()
			return
		}

		if userPriority < minPriority {
			response.Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin 要求管理员权限
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(RoleAdmin)
}

// RequireOperator 要求操作员或以上权限
func RequireOperator() gin.HandlerFunc {
	return RequireMinRole(RoleOperator)
}

// CanManageResource 检查是否可以管理资源
func CanManageResource(c *gin.Context, ownerID uint) bool {
	userRole := GetUserRole(c)
	userID := GetUserID(c)

	// 管理员可以管理所有资源
	if userRole == RoleAdmin {
		return true
	}

	// 操作员可以管理自己创建的资源
	if userRole == RoleOperator && userID == ownerID {
		return true
	}

	return false
}
