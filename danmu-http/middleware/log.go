package middleware

import (
	"danmu-http/internal/model"
	"danmu-http/logger"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		end := time.Now()
		// 执行时间
		latency := end.Sub(start)

		// 获取认证信息（如果存在）
		var email string
		if auth, exists := c.Get("auth"); exists && auth != nil {
			email = auth.(*model.Auth).Email
		}

		// 记录日志
		logger.Info().
			Int("status", c.Writer.Status()).
			Str("client_ip", c.ClientIP()).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Dur("latency", latency).
			Str("user_agent", c.Request.UserAgent()).
			Str("auth_email", email).
			Str("referer", c.Request.Referer()).
			Msg("HTTP Request")

		// 如果是错误状态码，额外记录错误日志
		if c.Writer.Status() >= 400 {
			logger.Error().
				Int("status", c.Writer.Status()).
				Str("client_ip", c.ClientIP()).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Dur("latency", latency).
				Str("user_agent", c.Request.UserAgent()).
				Str("user_email", email).
				Msg("HTTP Error")
		}
	}
}
