package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var Logger *zap.Logger

// InitLogger 初始化日志
func InitLogger(level, output string) error {
	var cfg zap.Config

	if output == "stdout" {
		cfg = zap.NewProductionConfig()
		cfg.OutputPaths = []string{"stdout"}
	} else {
		cfg = zap.NewProductionConfig()
		cfg.OutputPaths = []string{output}
	}

	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	var err error
	Logger, err = cfg.Build()
	if err != nil {
		return err
	}

	return nil
}

// sanitizeQuery 过滤敏感查询参数
func sanitizeQuery(query string) string {
	if query == "" {
		return ""
	}
	// 敏感参数列表
	sensitiveParams := []string{"password", "token", "secret", "key", "auth"}
	parts := strings.Split(query, "&")
	var sanitized []string
	for _, part := range parts {
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) == 2 {
			key := strings.ToLower(keyValue[0])
			isSensitive := false
			for _, s := range sensitiveParams {
				if strings.Contains(key, s) {
					isSensitive = true
					break
				}
			}
			if isSensitive {
				sanitized = append(sanitized, keyValue[0]+"=[REDACTED]")
			} else {
				sanitized = append(sanitized, part)
			}
		} else {
			sanitized = append(sanitized, part)
		}
	}
	return strings.Join(sanitized, "&")
}

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := sanitizeQuery(c.Request.URL.RawQuery)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if Logger != nil {
			Logger.Info("HTTP Request",
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("query", query),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("ip", c.ClientIP()),
			)
		}
	}
}

// ErrorLogger 错误日志中间件
func ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, err := range c.Errors {
			if Logger != nil {
				Logger.Error("Request Error",
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Error(err.Err),
				)
			}
		}
	}
}
