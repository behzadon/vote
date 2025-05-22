package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/behzadon/vote/internal/auth"
	"github.com/behzadon/vote/internal/domain"
	"github.com/behzadon/vote/internal/metrics"
	"github.com/behzadon/vote/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Handler struct {
	service     service.Service
	logger      *zap.Logger
	rateLimiter *RateLimiter
	authHandler *AuthHandler
}

func NewHandler(service service.Service, redis RedisClient, logger *zap.Logger, authHandler *AuthHandler) *Handler {
	return &Handler{
		service:     service,
		logger:      logger,
		rateLimiter: NewRateLimiter(redis, logger),
		authHandler: authHandler,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine, jwtManager *auth.JWTManager) {
	r.Use(metrics.MetricsMiddleware())

	r.POST("/api/auth/register", h.authHandler.Register)
	r.POST("/api/auth/login", h.authHandler.Login)
	r.GET("/api/polls/:id/stats", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.getPollStats)

	api := r.Group("/api")
	api.Use(auth.AuthMiddleware(jwtManager))
	{
		api.POST("/polls", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.createPoll)
		api.GET("/polls", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.getPollsForFeed)
		api.GET("/polls/:id", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.getPollByID)
		api.POST("/polls/:id/vote", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.voteOnPoll)
		api.POST("/polls/:id/skip", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.skipPoll)
		api.GET("/users/me/votes", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.getUserVotes)
		api.PUT("/users/me/votes/:voteId", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.updateVote)
		api.DELETE("/users/me/votes/:voteId", h.rateLimiter.RateLimit(), h.rateLimiter.BurstLimit(), h.deleteVote)
	}

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func (h *Handler) createPoll(c *gin.Context) {
	var req struct {
		Title   string   `json:"title" binding:"required"`
		Options []string `json:"options" binding:"required,min=2"`
		Tags    []string `json:"tags" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request body",
		})
		return
	}

	serviceReq := &domain.CreatePollRequest{
		Title:   req.Title,
		Options: req.Options,
		Tags:    req.Tags,
	}
	pollID, err := h.service.CreatePoll(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("failed to create poll",
			zap.Error(err),
			zap.String("title", req.Title),
		)
		switch {
		case errors.Is(err, domain.ErrInvalidInput):
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to create poll",
			})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"poll_id": pollID.String(),
	})
}

func (h *Handler) getPollsForFeed(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "user not authenticated",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid user id",
		})
		return
	}

	tag := c.Query("tag")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid page number",
		})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > domain.MaxPageSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid limit",
		})
		return
	}

	response, err := h.service.GetPollsForFeed(c.Request.Context(), userUUID, tag, page, limit)
	if err != nil {
		h.logger.Error("failed to get polls for feed",
			zap.Error(err),
			zap.String("userId", userUUID.String()),
			zap.String("tag", tag),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Failed to get polls",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"polls": response.Polls,
			"total": response.Total,
			"page":  response.Page,
			"limit": response.Limit,
		},
	})
}

func (h *Handler) getPollByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid poll id",
		})
		return
	}

	poll, err := h.service.GetPollByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get poll",
			zap.Error(err),
			zap.String("pollId", id.String()),
		)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "poll not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to get poll",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   poll,
	})
}

func (h *Handler) getPollStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid poll ID",
		})
		return
	}
	stats, err := h.service.GetPollStats(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("failed to get poll stats",
			zap.Error(err),
			zap.String("pollId", id.String()),
		)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "Poll not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to get poll stats",
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"poll_id": stats.PollID.String(),
			"votes":   stats.Votes,
		},
	})
}

type VoteOnPollRequest struct {
	UserID      string `json:"userId" binding:"required"`
	OptionIndex *int   `json:"optionIndex" binding:"required,min=0"`
}

func (h *Handler) voteOnPoll(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Error: "user not authenticated",
		})
		return
	}

	var req struct {
		OptionIndex *int `json:"optionIndex" binding:"required,min=0"`
	}
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid poll ID",
		})
		return
	}

	var rawBody []byte
	if c.Request.Body != nil {
		rawBody, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
		h.logger.Info("voteOnPoll: raw request body", zap.ByteString("rawBody", rawBody))

		var directReq map[string]interface{}
		if err := json.Unmarshal(rawBody, &directReq); err != nil {
			h.logger.Error("voteOnPoll: failed to unmarshal request body directly", zap.Error(err))
		} else {
			h.logger.Info("voteOnPoll: direct unmarshal result", zap.Any("directReq", directReq))
		}
	}

	if err := c.BindJSON(&req); err != nil {
		h.logger.Error("voteOnPoll: failed to bind JSON",
			zap.Error(err),
			zap.Any("req", req),
			zap.Any("rawBody", string(rawBody)),
			zap.Any("contentType", c.GetHeader("Content-Type")))
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request body",
		})
		return
	}

	h.logger.Info("voteOnPoll: successfully bound request", zap.Any("req", req))

	serviceReq := &domain.VoteRequest{
		UserID:      userID.(uuid.UUID),
		OptionIndex: *req.OptionIndex,
	}
	err = h.service.VoteOnPoll(c.Request.Context(), id, serviceReq)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrAlreadyVoted):
			h.logger.Info("user attempted to vote again on poll",
				zap.String("pollId", id.String()),
				zap.String("userId", serviceReq.UserID.String()),
			)
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
		case errors.Is(err, domain.ErrDailyVoteLimitExceeded):
			h.logger.Info("user exceeded daily vote limit",
				zap.String("pollId", id.String()),
				zap.String("userId", serviceReq.UserID.String()),
			)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
		case errors.Is(err, domain.ErrInvalidOption):
			h.logger.Error("invalid option selected for vote",
				zap.Error(err),
				zap.String("pollId", id.String()),
				zap.String("userId", serviceReq.UserID.String()),
				zap.Int("optionIndex", *req.OptionIndex),
			)
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
		case errors.Is(err, domain.ErrNotFound):
			h.logger.Error("poll not found for vote",
				zap.Error(err),
				zap.String("pollId", id.String()),
				zap.String("userId", serviceReq.UserID.String()),
			)
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "Poll not found",
			})
		default:
			h.logger.Error("failed to vote on poll",
				zap.Error(err),
				zap.String("pollId", id.String()),
				zap.String("userId", serviceReq.UserID.String()),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to vote on poll",
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func (h *Handler) skipPoll(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, domain.ErrorResponse{
			Error: "user not authenticated",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid poll ID",
		})
		return
	}

	serviceReq := &domain.SkipRequest{
		UserID: userID.(uuid.UUID),
	}
	err = h.service.SkipPoll(c.Request.Context(), id, serviceReq)
	if err != nil {
		h.logger.Error("failed to skip poll",
			zap.Error(err),
			zap.String("pollId", id.String()),
			zap.String("userId", serviceReq.UserID.String()),
		)
		switch {
		case errors.Is(err, domain.ErrAlreadySkipped):
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "Already skipped this poll",
			})
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "Poll not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to skip poll",
			})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func (h *Handler) getUserVotes(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "user not authenticated",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid user id",
		})
		return
	}

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = domain.DefaultPage
	}

	limitNum, err := strconv.Atoi(limit)
	if err != nil || limitNum < 1 || limitNum > domain.MaxPageSize {
		limitNum = domain.DefaultLimit
	}

	response, err := h.service.GetUserVotes(c.Request.Context(), userUUID, pageNum, limitNum)
	if err != nil {
		h.logger.Error("failed to get user votes",
			zap.Error(err),
			zap.String("userId", userUUID.String()),
		)
		switch {
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "user not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to get user votes",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   response,
	})
}

func (h *Handler) updateVote(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "user not authenticated",
		})
		return
	}

	voteIDStr := c.Param("voteId")
	voteID, err := uuid.Parse(voteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid vote id",
		})
		return
	}

	var req struct {
		OptionIndex int `json:"optionIndex" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid request body",
		})
		return
	}

	serviceReq := &domain.UpdateVoteRequest{
		UserID:      userID.(uuid.UUID),
		OptionIndex: req.OptionIndex,
	}

	err = h.service.UpdateVote(c.Request.Context(), voteID, serviceReq)
	if err != nil {
		h.logger.Error("failed to update vote",
			zap.Error(err),
			zap.String("voteId", voteID.String()),
			zap.String("userId", serviceReq.UserID.String()),
		)
		switch {
		case errors.Is(err, domain.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "unauthorized to update this vote",
			})
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "vote not found",
			})
		case errors.Is(err, domain.ErrInvalidOption):
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to update vote",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func (h *Handler) deleteVote(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "user not authenticated",
		})
		return
	}

	voteIDStr := c.Param("voteId")
	voteID, err := uuid.Parse(voteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid vote id",
		})
		return
	}

	err = h.service.DeleteVote(c.Request.Context(), voteID, userID.(uuid.UUID))
	if err != nil {
		h.logger.Error("failed to delete vote",
			zap.Error(err),
			zap.String("voteId", voteID.String()),
			zap.String("userId", userID.(uuid.UUID).String()),
		)
		switch {
		case errors.Is(err, domain.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "unauthorized to delete this vote",
			})
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "vote not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "failed to delete vote",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

func (h *Handler) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
