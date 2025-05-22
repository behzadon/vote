package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/behzadon/vote/internal/auth"
	"github.com/behzadon/vote/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreatePoll(ctx context.Context, req *domain.CreatePollRequest) (uuid.UUID, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockService) GetPollByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Poll), args.Error(1)
}

func (m *MockService) GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) (*domain.PollFeedResponse, error) {
	args := m.Called(ctx, userID, tag, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PollFeedResponse), args.Error(1)
}

func (m *MockService) GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	args := m.Called(ctx, pollID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PollStats), args.Error(1)
}

func (m *MockService) VoteOnPoll(ctx context.Context, pollID uuid.UUID, req *domain.VoteRequest) error {
	args := m.Called(ctx, pollID, req)
	return args.Error(0)
}

func (m *MockService) SkipPoll(ctx context.Context, pollID uuid.UUID, req *domain.SkipRequest) error {
	args := m.Called(ctx, pollID, req)
	return args.Error(0)
}

func (m *MockService) CreateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockService) UpdateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) GetUserVotes(ctx context.Context, userID uuid.UUID, page, limit int) (*domain.UserVotesResponse, error) {
	args := m.Called(ctx, userID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserVotesResponse), args.Error(1)
}

func (m *MockService) DeleteVote(ctx context.Context, voteID uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, voteID, userID)
	return args.Error(0)
}

func (m *MockService) UpdateVote(ctx context.Context, voteID uuid.UUID, req *domain.UpdateVoteRequest) error {
	args := m.Called(ctx, voteID, req)
	return args.Error(0)
}

type MockRedis struct {
	*redis.Client
	counters map[string]int64
	windows  map[string]int64
}

func NewMockRedis() *MockRedis {
	return &MockRedis{
		Client:   redis.NewClient(&redis.Options{}),
		counters: make(map[string]int64),
		windows:  make(map[string]int64),
	}
}

func (m *MockRedis) Incr(ctx context.Context, key string) *redis.IntCmd {
	m.counters[key]++
	return redis.NewIntResult(m.counters[key], nil)
}

func (m *MockRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	if strings.HasSuffix(key, ":count") {
		if count, exists := m.counters[key]; exists {
			return redis.NewStringResult(strconv.FormatInt(count, 10), nil)
		}
		return redis.NewStringResult("0", nil)
	}
	if strings.HasSuffix(key, ":window") {
		if window, exists := m.windows[key]; exists {
			return redis.NewStringResult(strconv.FormatInt(window, 10), nil)
		}
		return redis.NewStringResult(strconv.FormatInt(time.Now().Unix(), 10), nil)
	}
	return redis.NewStringResult("", redis.Nil)
}

func (m *MockRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	if strings.HasSuffix(key, ":window") {
		if val, ok := value.(int64); ok {
			m.windows[key] = val
		}
	}
	return redis.NewStatusResult("OK", nil)
}

func (m *MockRedis) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	return redis.NewBoolResult(true, nil)
}

func (m *MockRedis) Pipeline() redis.Pipeliner {
	return &MockPipeline{mockRedis: m}
}

type MockPipeline struct {
	redis.Pipeliner
	mockRedis *MockRedis
}

func (m *MockPipeline) Exec(ctx context.Context) ([]redis.Cmder, error) {
	return nil, nil
}

func (m *MockPipeline) Get(ctx context.Context, key string) *redis.StringCmd {
	return m.mockRedis.Get(ctx, key)
}

func (m *MockPipeline) Incr(ctx context.Context, key string) *redis.IntCmd {
	return m.mockRedis.Incr(ctx, key)
}

func (m *MockPipeline) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return m.mockRedis.Set(ctx, key, value, expiration)
}

func setupTest(t *testing.T) (*gin.Engine, *MockService, *Handler, *AuthHandler, *auth.JWTManager) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mockService := new(MockService)
	logger, _ := zap.NewDevelopment()

	mockRedis := NewMockRedis()

	jwtManager := auth.NewJWTManager("test-secret", 24*time.Hour)
	authHandler := NewAuthHandler(mockService, jwtManager, logger)
	handler := NewHandler(mockService, mockRedis, logger, authHandler)

	testAuthMiddleware := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "unauthorized",
			})
			c.Abort()
			return
		}

		token := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "invalid token",
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}

	api := r.Group("/api")
	api.Use(testAuthMiddleware)
	{
		api.POST("/polls", handler.rateLimiter.RateLimit(), handler.rateLimiter.BurstLimit(), handler.createPoll)
		api.GET("/polls", handler.rateLimiter.RateLimit(), handler.rateLimiter.BurstLimit(), handler.getPollsForFeed)
		api.GET("/polls/:id", handler.rateLimiter.RateLimit(), handler.rateLimiter.BurstLimit(), handler.getPollByID)
		api.POST("/polls/:id/vote", handler.rateLimiter.RateLimit(), handler.rateLimiter.BurstLimit(), handler.voteOnPoll)
		api.POST("/polls/:id/skip", handler.rateLimiter.RateLimit(), handler.rateLimiter.BurstLimit(), handler.skipPoll)
	}

	r.POST("/api/auth/register", authHandler.Register)
	r.POST("/api/auth/login", authHandler.Login)
	r.GET("/api/polls/:id/stats", handler.rateLimiter.RateLimit(), handler.rateLimiter.BurstLimit(), handler.getPollStats)

	return r, mockService, handler, authHandler, jwtManager
}

func TestCreatePoll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, mockService, _, _, jwtManager := setupTest(t)
		userID := uuid.New()
		token, _ := jwtManager.GenerateToken(&domain.User{ID: userID})

		req := domain.CreatePollRequest{
			Title:   "Test Poll",
			Options: []string{"Option 1", "Option 2"},
			Tags:    []string{"test"},
		}

		pollID := uuid.New()
		mockService.On("CreatePoll", mock.Anything, &req).Return(pollID, nil)

		w := httptest.NewRecorder()
		body, _ := json.Marshal(req)
		request, _ := http.NewRequest("POST", "/api/polls", bytes.NewBuffer(body))
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		assert.Equal(t, "success", response["status"])
		assert.Equal(t, pollID.String(), response["poll_id"])
	})

	t.Run("unauthorized", func(t *testing.T) {
		r, _, _, _, _ := setupTest(t)
		req := domain.CreatePollRequest{
			Title:   "Test Poll",
			Options: []string{"Option 1", "Option 2"},
			Tags:    []string{"test"},
		}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(req)
		request, _ := http.NewRequest("POST", "/api/polls", bytes.NewBuffer(body))
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestVoteOnPoll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, mockService, _, _, jwtManager := setupTest(t)
		userID := uuid.New()
		token, _ := jwtManager.GenerateToken(&domain.User{ID: userID})
		pollID := uuid.New()

		req := domain.VoteRequest{
			UserID:      userID,
			OptionIndex: 0,
		}

		mockService.On("VoteOnPoll", mock.Anything, pollID, &req).Return(nil)

		w := httptest.NewRecorder()
		body, _ := json.Marshal(req)
		request, _ := http.NewRequest("POST", "/api/polls/"+pollID.String()+"/vote", bytes.NewBuffer(body))
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusOK, w.Code)
		var result map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, "success", result["status"])
	})

	t.Run("already voted", func(t *testing.T) {
		r, mockService, _, _, jwtManager := setupTest(t)
		userID := uuid.New()
		token, _ := jwtManager.GenerateToken(&domain.User{ID: userID})
		pollID := uuid.New()

		req := domain.VoteRequest{
			UserID:      userID,
			OptionIndex: 0,
		}

		mockService.On("VoteOnPoll", mock.Anything, pollID, &req).Return(domain.ErrAlreadyVoted)

		w := httptest.NewRecorder()
		body, _ := json.Marshal(req)
		request, _ := http.NewRequest("POST", "/api/polls/"+pollID.String()+"/vote", bytes.NewBuffer(body))
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("unauthorized", func(t *testing.T) {
		r, _, _, _, _ := setupTest(t)
		pollID := uuid.New()
		req := domain.VoteRequest{
			OptionIndex: 0,
		}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(req)
		request, _ := http.NewRequest("POST", "/api/polls/"+pollID.String()+"/vote", bytes.NewBuffer(body))
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetPollStats(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, mockService, _, _, _ := setupTest(t)
		pollID := uuid.New()

		stats := &domain.PollStats{
			PollID: pollID,
			Votes: []domain.OptionStats{
				{Option: "Option 1", Count: 10},
				{Option: "Option 2", Count: 5},
			},
		}

		mockService.On("GetPollStats", mock.Anything, mock.MatchedBy(func(id uuid.UUID) bool {
			return id == pollID
		})).Return(stats, nil).Once()

		w := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/api/polls/"+pollID.String()+"/stats", nil)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["status"])

		data, ok := response["data"].(map[string]interface{})
		assert.True(t, ok, "data field should be a map")
		assert.Equal(t, pollID.String(), data["poll_id"])

		votes, ok := data["votes"].([]interface{})
		assert.True(t, ok, "votes field should be an array")
		assert.Equal(t, 2, len(votes))

		vote1, ok := votes[0].(map[string]interface{})
		assert.True(t, ok, "first vote should be a map")
		assert.Equal(t, "Option 1", vote1["option"])
		assert.Equal(t, float64(10), vote1["count"])

		vote2, ok := votes[1].(map[string]interface{})
		assert.True(t, ok, "second vote should be a map")
		assert.Equal(t, "Option 2", vote2["option"])
		assert.Equal(t, float64(5), vote2["count"])

		mockService.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		r, mockService, _, _, _ := setupTest(t)
		pollID := uuid.New()

		mockService.On("GetPollStats", mock.Anything, mock.MatchedBy(func(id uuid.UUID) bool {
			return id == pollID
		})).Return(nil, domain.ErrNotFound).Once()

		w := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/api/polls/"+pollID.String()+"/stats", nil)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response["status"])
		assert.Equal(t, "Poll not found", response["message"])

		mockService.AssertExpectations(t)
	})

	t.Run("invalid poll ID", func(t *testing.T) {
		r, _, _, _, _ := setupTest(t)
		w := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/api/polls/invalid-id/stats", nil)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "error", response["status"])
		assert.Equal(t, "Invalid poll ID", response["message"])
	})
}

func TestGetPollsForFeed(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, mockService, _, _, jwtManager := setupTest(t)
		userID := uuid.New()
		token, _ := jwtManager.GenerateToken(&domain.User{ID: userID})

		pollID := uuid.New()
		option1ID := uuid.New()
		option2ID := uuid.New()
		response := &domain.PollFeedResponse{
			Polls: []domain.Poll{
				{
					ID:    pollID,
					Title: "Test Poll",
					Options: []domain.Option{
						{ID: option1ID, OptionText: "Option 1"},
						{ID: option2ID, OptionText: "Option 2"},
					},
					Tags:      []string{"test"},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			Total: 1,
			Page:  1,
			Limit: 10,
		}

		mockService.On("GetPollsForFeed", mock.Anything, userID, "", 1, 10).Return(response, nil)

		w := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/api/polls?page=1&limit=10", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusOK, w.Code)
		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, "success", result["status"])

		data, ok := result["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, float64(1), data["total"])
		assert.Equal(t, float64(1), data["page"])
		assert.Equal(t, float64(10), data["limit"])

		polls, ok := data["polls"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 1, len(polls))

		poll, ok := polls[0].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, pollID.String(), poll["id"])
		assert.Equal(t, "Test Poll", poll["title"])
	})

	t.Run("unauthorized", func(t *testing.T) {
		r, _, _, _, _ := setupTest(t)
		w := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/api/polls", nil)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetPollByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, mockService, _, _, jwtManager := setupTest(t)
		userID := uuid.New()
		token, _ := jwtManager.GenerateToken(&domain.User{ID: userID})
		pollID := uuid.New()
		option1ID := uuid.New()
		option2ID := uuid.New()

		poll := &domain.Poll{
			ID:    pollID,
			Title: "Test Poll",
			Options: []domain.Option{
				{ID: option1ID, OptionText: "Option 1"},
				{ID: option2ID, OptionText: "Option 2"},
			},
			Tags:      []string{"test"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockService.On("GetPollByID", mock.Anything, pollID).Return(poll, nil)

		w := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/api/polls/"+pollID.String(), nil)
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusOK, w.Code)
		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, "success", result["status"])

		data, ok := result["data"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, pollID.String(), data["id"])
		assert.Equal(t, "Test Poll", data["title"])
	})

	t.Run("not found", func(t *testing.T) {
		r, mockService, _, _, jwtManager := setupTest(t)
		userID := uuid.New()
		token, _ := jwtManager.GenerateToken(&domain.User{ID: userID})
		pollID := uuid.New()

		mockService.On("GetPollByID", mock.Anything, pollID).Return(nil, domain.ErrNotFound)

		w := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/api/polls/"+pollID.String(), nil)
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var result map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, "error", result["status"])
		assert.Equal(t, "poll not found", result["message"])
	})

	t.Run("unauthorized", func(t *testing.T) {
		r, _, _, _, _ := setupTest(t)
		pollID := uuid.New()
		w := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/api/polls/"+pollID.String(), nil)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSkipPoll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, mockService, _, _, jwtManager := setupTest(t)
		userID := uuid.New()
		token, _ := jwtManager.GenerateToken(&domain.User{ID: userID})
		pollID := uuid.New()

		req := domain.SkipRequest{
			UserID: userID,
		}

		mockService.On("SkipPoll", mock.Anything, pollID, &req).Return(nil)

		w := httptest.NewRecorder()
		body, _ := json.Marshal(req)
		request, _ := http.NewRequest("POST", "/api/polls/"+pollID.String()+"/skip", bytes.NewBuffer(body))
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusOK, w.Code)
		var result map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &result)
		assert.Equal(t, "success", result["status"])
	})

	t.Run("already skipped", func(t *testing.T) {
		r, mockService, _, _, jwtManager := setupTest(t)
		userID := uuid.New()
		token, _ := jwtManager.GenerateToken(&domain.User{ID: userID})
		pollID := uuid.New()

		req := domain.SkipRequest{
			UserID: userID,
		}

		mockService.On("SkipPoll", mock.Anything, pollID, &req).Return(domain.ErrAlreadySkipped)

		w := httptest.NewRecorder()
		body, _ := json.Marshal(req)
		request, _ := http.NewRequest("POST", "/api/polls/"+pollID.String()+"/skip", bytes.NewBuffer(body))
		request.Header.Set("Authorization", "Bearer "+token)
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("unauthorized", func(t *testing.T) {
		r, _, _, _, _ := setupTest(t)
		pollID := uuid.New()
		req := domain.SkipRequest{}

		w := httptest.NewRecorder()
		body, _ := json.Marshal(req)
		request, _ := http.NewRequest("POST", "/api/polls/"+pollID.String()+"/skip", bytes.NewBuffer(body))
		r.ServeHTTP(w, request)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case []string:
		return vv
	case []interface{}:
		out := make([]string, len(vv))
		for i, val := range vv {
			out[i], _ = val.(string)
		}
		return out
	}
	return nil
}
