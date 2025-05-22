package notification

import (
	"context"

	"go.uber.org/zap"
)

type MockNotificationService struct {
	Logger *zap.Logger
}

func (s *MockNotificationService) SendNotification(ctx context.Context, userID string, title, message string) error {
	s.Logger.Info("Mock notification sent",
		zap.String("user_id", userID),
		zap.String("title", title),
		zap.String("message", message),
	)
	return nil
}
