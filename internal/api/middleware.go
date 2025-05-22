package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	DefaultRateLimit     = 1000
	DefaultRateWindow    = 60
	DefaultBurstLimit    = 500
	DefaultCleanupWindow = 3600
)

type RateLimiter struct {
	redis  RedisClient
	logger *zap.Logger
}

func NewRateLimiter(redis RedisClient, logger *zap.Logger) *RateLimiter {
	return &RateLimiter{
		redis:  redis,
		logger: logger,
	}
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/polls/") && strings.HasSuffix(c.Request.URL.Path, "/stats") {
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			if c.Request.Method == http.MethodGet {
				userID = c.Query("userId")
			} else if c.Request.Method == http.MethodPost {
				if c.Request.Body != nil {
					body, err := io.ReadAll(c.Request.Body)
					if err != nil {
						rl.logger.Error("failed to read request body in rate limit middleware",
							zap.Error(err),
						)
					} else {
						c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
					}
				}
				c.Next()
				return
			}
		}

		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "User ID is required",
			})
			c.Abort()
			return
		}

		userIDStr := ""
		switch v := userID.(type) {
		case string:
			userIDStr = v
		case uuid.UUID:
			userIDStr = v.String()
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid user ID type",
			})
			c.Abort()
			return
		}

		key := "rate_limit:" + userIDStr + ":" + c.Request.URL.Path

		ctx := c.Request.Context()
		pipe := rl.redis.Pipeline()
		now := time.Now().Unix()
		windowKey := key + ":window"
		countKey := key + ":count"

		getCount := pipe.Get(ctx, countKey)
		getWindow := pipe.Get(ctx, windowKey)

		if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
			rl.logger.Error("failed to get rate limit info",
				zap.Error(err),
				zap.String("user_id", userIDStr),
				zap.String("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Rate limit check failed",
			})
			c.Abort()
			return
		}

		count := 0
		window := now
		if countStr, err := getCount.Result(); err == nil {
			if count, err = strconv.Atoi(countStr); err != nil {
				rl.logger.Error("failed to parse count",
					zap.Error(err),
					zap.String("count", countStr),
				)
			}
		}
		if windowStr, err := getWindow.Result(); err == nil {
			if window, err = strconv.ParseInt(windowStr, 10, 64); err != nil {
				rl.logger.Error("failed to parse window",
					zap.Error(err),
					zap.String("window", windowStr),
				)
			}
		}

		if now-window >= DefaultRateWindow {
			count = 0
			window = now
		}

		if count >= DefaultRateLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		pipe = rl.redis.Pipeline()
		pipe.Incr(ctx, countKey)
		pipe.Set(ctx, windowKey, window, DefaultCleanupWindow*time.Second)
		if _, err := pipe.Exec(ctx); err != nil {
			rl.logger.Error("failed to update rate limit",
				zap.Error(err),
				zap.String("user_id", userIDStr),
				zap.String("path", c.Request.URL.Path),
			)
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(DefaultRateLimit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(DefaultRateLimit-count-1))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(window+DefaultRateWindow, 10))

		c.Next()
	}
}

func (rl *RateLimiter) BurstLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/polls/") && strings.HasSuffix(c.Request.URL.Path, "/stats") {
			c.Next()
			return
		}

		userID, exists := c.Get("user_id")
		if !exists {
			if c.Request.Method == http.MethodGet {
				userID = c.Query("userId")
			} else if c.Request.Method == http.MethodPost {
				if c.Request.Body != nil {
					body, err := io.ReadAll(c.Request.Body)
					if err != nil {
						rl.logger.Error("failed to read request body in burst limit middleware",
							zap.Error(err),
						)
					} else {
						c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
					}
				}
				c.Next()
				return
			}
		}

		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "User ID is required",
			})
			c.Abort()
			return
		}

		userIDStr := ""
		switch v := userID.(type) {
		case string:
			userIDStr = v
		case uuid.UUID:
			userIDStr = v.String()
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Invalid user ID type",
			})
			c.Abort()
			return
		}

		key := "burst_limit:" + userIDStr + ":" + c.Request.URL.Path
		ctx := c.Request.Context()
		count, err := rl.redis.Incr(ctx, key).Result()
		if err != nil {
			rl.logger.Error("failed to increment burst limit",
				zap.Error(err),
				zap.String("user_id", userIDStr),
				zap.String("path", c.Request.URL.Path),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Burst limit check failed",
			})
			c.Abort()
			return
		}

		if count == 1 {
			if err := rl.redis.Expire(ctx, key, time.Second).Err(); err != nil {
				rl.logger.Error("failed to set burst limit expiry",
					zap.Error(err),
					zap.String("user_id", userIDStr),
					zap.String("path", c.Request.URL.Path),
				)
			}
		}

		if count > DefaultBurstLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": "Burst limit exceeded",
			})
			c.Abort()
			return
		}

		c.Header("X-BurstLimit-Limit", strconv.Itoa(DefaultBurstLimit))
		c.Header("X-BurstLimit-Remaining", strconv.FormatInt(DefaultBurstLimit-count, 10))

		c.Next()
	}
}
