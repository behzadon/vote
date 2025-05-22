package api

import (
	"net/http"
	"time"

	"github.com/behzadon/vote/internal/auth"
	"github.com/behzadon/vote/internal/domain"
	"github.com/behzadon/vote/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthHandler struct {
	service    service.Service
	jwtManager auth.JWTManagerInterface
	logger     *zap.Logger
}

func NewAuthHandler(service service.Service, jwtManager auth.JWTManagerInterface, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		service:    service,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

func (h *AuthHandler) RegisterRoutes(r *gin.Engine) {
	auth := r.Group("/api/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.GET("/profile", h.AuthMiddleware(), h.GetProfile)
	}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	user := &domain.User{
		Email:    req.Email,
		Password: req.Password,
		Username: req.Username,
	}

	if err := h.service.CreateUser(c.Request.Context(), user); err != nil {
		if err == domain.ErrEmailAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}
		h.logger.Error("failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "failed to create user",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}

	user, err := h.service.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if err == domain.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
			return
		}
		h.logger.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "failed to login",
		})
		return
	}

	token, err := h.jwtManager.GenerateToken(user)
	if err != nil {
		h.logger.Error("failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"token":  token,
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "unauthorized",
		})
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "user not found",
			})
			return
		}
		h.logger.Error("failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "failed to get user profile",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"id":        user.ID.String(),
			"email":     user.Email,
			"username":  user.Username,
			"createdAt": user.CreatedAt.Format(time.RFC3339),
			"updatedAt": user.UpdatedAt.Format(time.RFC3339),
		},
	})
}

func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "unauthorized",
			})
			c.Abort()
			return
		}

		claims, err := h.jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "invalid token",
			})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Next()
	}
}
