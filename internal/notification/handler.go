package notification

import (
	"context"

	"github.com/behzadon/vote/internal/domain"
	"github.com/behzadon/vote/internal/storage/events"
	"go.uber.org/zap"
)

type NotificationService interface {
	SendNotification(ctx context.Context, userID string, title, message string) error
}

type NotificationHandler struct {
	notificationService NotificationService
	logger              *zap.Logger
}

func NewNotificationHandler(notificationService NotificationService, logger *zap.Logger) events.EventHandler {
	return &NotificationHandler{
		notificationService: notificationService,
		logger:              logger,
	}
}

func (h *NotificationHandler) HandlePollCreated(_ context.Context, poll *domain.Poll) error {

	for _, tag := range poll.Tags {
		h.logger.Info("Would notify users following tag",
			zap.String("tag", tag),
			zap.String("poll_id", poll.ID.String()),
			zap.String("poll_title", poll.Title),
		)
	}

	return nil
}

func (h *NotificationHandler) HandlePollVoted(ctx context.Context, vote *domain.Vote) error {
	h.logger.Info("Would notify poll creator about new vote",
		zap.String("poll_id", vote.PollID.String()),
		zap.String("voter_id", vote.UserID.String()),
	)

	return nil
}

func (h *NotificationHandler) HandlePollSkipped(ctx context.Context, skip *domain.Skip) error {
	h.logger.Info("Poll skipped",
		zap.String("poll_id", skip.PollID.String()),
		zap.String("user_id", skip.UserID.String()),
		zap.Time("timestamp", skip.CreatedAt),
	)

	return nil
}
