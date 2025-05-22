package events

import (
	"context"

	"github.com/behzadon/vote/internal/domain"
)

type Publisher interface {
	PublishPollCreated(ctx context.Context, poll *domain.Poll) error
	PublishPollVoted(ctx context.Context, vote *domain.Vote) error
	PublishPollSkipped(ctx context.Context, skip *domain.Skip) error
	Close() error
}
