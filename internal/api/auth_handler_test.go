package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/behzadon/vote/internal/auth"
	"github.com/behzadon/vote/internal/domain"
	"github.com/behzadon/vote/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestAuthHandler_Register(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(service.MockService)
	mockJWTManager := new(auth.MockJWTManager)
	logger, _ := zap.NewDevelopment()
	handler := NewAuthHandler(mockService, mockJWTManager, logger)

	tests := []struct {
		name           string
		request        domain.RegisterRequest
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful registration",
			request: domain.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Username: "testuser",
			},
			mockSetup: func() {
				mockService.On("CreateUser", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
					return user.Email == "test@example.com" && user.Username == "testuser"
				})).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			expectedBody: map[string]interface{}{
				"status": "success",
			},
		},
		{
			name: "email already exists",
			request: domain.RegisterRequest{
				Email:    "existing@example.com",
				Password: "password123",
				Username: "existinguser",
			},
			mockSetup: func() {
				mockService.On("CreateUser", mock.Anything, mock.Anything).Return(domain.ErrEmailAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedBody: map[string]interface{}{
				"status":  "error",
				"message": domain.ErrEmailAlreadyExists.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/auth/register", handler.Register)

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, response)
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(service.MockService)
	mockJWTManager := new(auth.MockJWTManager)
	logger, _ := zap.NewDevelopment()
	handler := NewAuthHandler(mockService, mockJWTManager, logger)

	userID := uuid.New()
	user := &domain.User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "testuser",
		Password: "hashedpassword",
	}

	tests := []struct {
		name           string
		request        domain.LoginRequest
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "successful login",
			request: domain.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func() {
				mockService.On("GetUserByEmail", mock.Anything, "test@example.com").Return(user, nil)
				mockJWTManager.On("GenerateToken", user).Return("test-token", nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status": "success",
				"token":  "test-token",
			},
		},
		{
			name: "invalid credentials",
			request: domain.LoginRequest{
				Email:    "wrong@example.com",
				Password: "wrongpass",
			},
			mockSetup: func() {
				mockService.On("GetUserByEmail", mock.Anything, "wrong@example.com").Return(nil, domain.ErrInvalidCredentials)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"status":  "error",
				"message": domain.ErrInvalidCredentials.Error(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router := gin.New()
			router.POST("/api/auth/login", handler.Login)

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, response)
		})
	}
}

func TestAuthHandler_GetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(service.MockService)
	mockJWTManager := new(auth.MockJWTManager)
	logger, _ := zap.NewDevelopment()
	handler := NewAuthHandler(mockService, mockJWTManager, logger)

	userID := uuid.New()
	user := &domain.User{
		ID:        userID,
		Email:     "test@example.com",
		Username:  "testuser",
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name           string
		userID         uuid.UUID
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:   "successful profile retrieval",
			userID: userID,
			mockSetup: func() {
				mockService.On("GetUserByID", mock.Anything, userID).Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"id":        user.ID.String(),
					"email":     user.Email,
					"username":  user.Username,
					"createdAt": user.CreatedAt.Format(time.RFC3339),
					"updatedAt": user.UpdatedAt.Format(time.RFC3339),
				},
			},
		},
		{
			name:   "user not found",
			userID: uuid.New(),
			mockSetup: func() {
				mockService.On("GetUserByID", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedBody: map[string]interface{}{
				"status":  "error",
				"message": "user not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil)
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/auth/profile", func(c *gin.Context) {
				c.Set("userID", tt.userID)
				handler.GetProfile(c)
			})

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBody, response)
		})
	}
}

func TestAuthHandler_AuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockService := new(service.MockService)
	mockJWTManager := new(auth.MockJWTManager)
	logger, _ := zap.NewDevelopment()
	handler := NewAuthHandler(mockService, mockJWTManager, logger)

	userID := uuid.New()
	claims := &auth.Claims{
		UserID: userID,
	}

	tests := []struct {
		name           string
		token          string
		mockSetup      func()
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name:  "valid token",
			token: "valid-token",
			mockSetup: func() {
				mockJWTManager.On("ValidateToken", "valid-token").Return(claims, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status": "success",
			},
		},
		{
			name:  "missing token",
			token: "",
			mockSetup: func() {
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"status":  "error",
				"message": "unauthorized",
			},
		},
		{
			name:  "invalid token",
			token: "invalid-token",
			mockSetup: func() {
				mockJWTManager.On("ValidateToken", "invalid-token").Return(nil, auth.ErrInvalidToken)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: map[string]interface{}{
				"status":  "error",
				"message": "invalid token",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest(http.MethodGet, "/api/auth/profile", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			w := httptest.NewRecorder()

			router := gin.New()
			router.GET("/api/auth/profile", handler.AuthMiddleware(), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "success"})
			})

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != nil {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}
		})
	}
}
