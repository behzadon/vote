package auth

import (
	"github.com/behzadon/vote/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockJWTManager struct {
	mock.Mock
}

func (m *MockJWTManager) GenerateToken(user *domain.User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func (m *MockJWTManager) ValidateToken(token string) (*Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Claims), args.Error(1)
}
