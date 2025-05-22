package logging

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Logger struct {
	zapLogger *zap.Logger
}

func NewLogger(zapLogger *zap.Logger) *Logger {
	return &Logger{
		zapLogger: zapLogger,
	}
}

func (l *Logger) GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		l.zapLogger.Info("incoming request",
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.String("method", method),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("error", errorMessage),
			zap.String("user-agent", c.Request.UserAgent()),
		)

		if gin.Mode() == gin.DebugMode {
			fmt.Printf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n",
				end.Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				method,
				path,
			)
		}
	}
}

func (l *Logger) Error(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	l.zapLogger.Error(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zapLogger.Info(msg, fields...)
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zapLogger.Debug(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zapLogger.Warn(msg, fields...)
}

func (l *Logger) Fatal(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	l.zapLogger.Fatal(msg, fields...)
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		zapLogger: l.zapLogger.With(fields...),
	}
}
