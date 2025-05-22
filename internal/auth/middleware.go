package auth

import (
	"net/http"
	"strings"

	"github.com/behzadon/vote/internal/domain"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := zap.L().With(
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.String("ip", c.ClientIP()),
			zap.Strings("headers", c.Request.Header["Authorization"]),
		)

		logger.Info("auth middleware: processing request")

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Info("auth middleware: missing authorization header")
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
				Error: "authorization header is required",
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Info("auth middleware: invalid authorization header format",
				zap.String("header", authHeader),
				zap.Int("parts", len(parts)),
				zap.Strings("parts", parts),
			)
			c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
				Error: "invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := parts[1]
		logger.Info("auth middleware: validating token",
			zap.String("token_prefix", token[:10]+"..."),
		)

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			logger.Info("auth middleware: token validation failed",
				zap.Error(err),
				zap.String("token_prefix", token[:10]+"..."),
			)
			status := http.StatusUnauthorized
			if err == ErrExpiredToken {
				status = http.StatusForbidden
			}
			c.JSON(status, domain.ErrorResponse{
				Error: err.Error(),
			})
			c.Abort()
			return
		}

		logger.Info("auth middleware: token validated successfully",
			zap.String("user_id", claims.UserID.String()),
			zap.String("username", claims.Username),
		)

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
