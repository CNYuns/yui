package middleware

import (
	"sync"
	"time"

	"y-ui/internal/database"
	"y-ui/internal/models"

	"github.com/gin-gonic/gin"
)

// CORS 跨域中间件 - 生产环境友好版本
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 允许的来源：
		// 1. 同源请求（Origin 为空）
		// 2. localhost 开发环境
		// 3. 127.0.0.1 开发环境
		// 4. 与 Host 相同的 Origin（生产环境）
		allowed := false
		if origin == "" {
			allowed = true
		} else {
			host := c.Request.Host
			// 检查是否是开发环境或同源
			allowedOrigins := []string{
				"http://localhost:3000",
				"http://127.0.0.1:3000",
				"http://localhost:8080",
				"http://127.0.0.1:8080",
				"http://" + host,
				"https://" + host,
			}
			for _, ao := range allowedOrigins {
				if origin == ao {
					allowed = true
					break
				}
			}
		}

		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Security 安全相关头部
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		c.Next()
	}
}

// rateLimitClient 速率限制客户端信息
type rateLimitClient struct {
	count     int
	lastSeen  time.Time
	blockedAt time.Time
}

// RateLimitStore 线程安全的速率限制存储
type RateLimitStore struct {
	clients map[string]*rateLimitClient
	mu      sync.RWMutex
}

// NewRateLimitStore 创建速率限制存储
func NewRateLimitStore() *RateLimitStore {
	store := &RateLimitStore{
		clients: make(map[string]*rateLimitClient),
	}
	// 启动清理协程
	go store.cleanup()
	return store
}

// cleanup 定期清理过期记录
func (s *RateLimitStore) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for ip, client := range s.clients {
			if now.Sub(client.lastSeen) > 30*time.Minute {
				delete(s.clients, ip)
			}
		}
		s.mu.Unlock()
	}
}

var loginRateLimitStore = NewRateLimitStore()

// RateLimitLogin 登录专用速率限制
func RateLimitLogin(maxAttempts int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		loginRateLimitStore.mu.Lock()
		defer loginRateLimitStore.mu.Unlock()

		client, exists := loginRateLimitStore.clients[ip]
		now := time.Now()

		if !exists {
			loginRateLimitStore.clients[ip] = &rateLimitClient{count: 1, lastSeen: now}
			c.Next()
			return
		}

		// 检查是否在封禁期
		if !client.blockedAt.IsZero() && now.Sub(client.blockedAt) < 15*time.Minute {
			c.AbortWithStatusJSON(429, gin.H{
				"code": 429,
				"msg":  "登录尝试次数过多，请15分钟后再试",
			})
			return
		}

		// 窗口期已过，重置计数
		if now.Sub(client.lastSeen) > window {
			client.count = 1
			client.lastSeen = now
			client.blockedAt = time.Time{}
			c.Next()
			return
		}

		client.count++
		client.lastSeen = now

		if client.count > maxAttempts {
			client.blockedAt = now
			c.AbortWithStatusJSON(429, gin.H{
				"code": 429,
				"msg":  "登录尝试次数过多，请15分钟后再试",
			})
			return
		}

		c.Next()
	}
}

// InitAdminProtection 保护初始化管理员接口
func InitAdminProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否已存在管理员
		var count int64
		database.DB.Model(&models.User{}).Count(&count)
		if count > 0 {
			c.AbortWithStatusJSON(403, gin.H{
				"code": 403,
				"msg":  "管理员已存在，此接口已禁用",
			})
			return
		}
		c.Next()
	}
}

// RateLimit 通用请求限制（带自动清理）
func RateLimit(maxRequests int, window time.Duration) gin.HandlerFunc {
	store := NewRateLimitStore()

	return func(c *gin.Context) {
		ip := c.ClientIP()

		store.mu.Lock()
		defer store.mu.Unlock()

		client, exists := store.clients[ip]
		now := time.Now()

		if !exists {
			store.clients[ip] = &rateLimitClient{count: 1, lastSeen: now}
			c.Next()
			return
		}

		if now.Sub(client.lastSeen) > window {
			client.count = 1
			client.lastSeen = now
		} else {
			client.count++
			if client.count > maxRequests {
				c.AbortWithStatusJSON(429, gin.H{
					"code": 429,
					"msg":  "请求过于频繁，请稍后再试",
				})
				return
			}
		}

		c.Next()
	}
}
