package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/behzadon/vote/internal/domain"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Publisher interface {
	PublishPollCreated(ctx context.Context, poll *domain.Poll) error
	PublishPollVoted(ctx context.Context, vote *domain.Vote) error
	PublishPollVoteUpdated(ctx context.Context, vote *domain.Vote) error
	PublishPollVoteDeleted(ctx context.Context, vote *domain.Vote) error
	PublishPollSkipped(ctx context.Context, skip *domain.Skip) error
	Close() error
}

type RedisPublisher struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisPublisher(client *redis.Client, logger *zap.Logger) *RedisPublisher {
	return &RedisPublisher{
		client: client,
		logger: logger,
	}
}

func (p *RedisPublisher) PublishPollCreated(ctx context.Context, poll *domain.Poll) error {
	event := struct {
		Type string       `json:"type"`
		Data *domain.Poll `json:"data"`
	}{
		Type: "poll.created",
		Data: poll,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal poll created event: %w", err)
	}

	if err := p.client.Publish(ctx, "events", data).Err(); err != nil {
		return fmt.Errorf("publish poll created event: %w", err)
	}

	p.logger.Info("published poll created event",
		zap.String("poll_id", poll.ID.String()),
		zap.String("title", poll.Title),
	)

	return nil
}

func (p *RedisPublisher) PublishPollVoted(ctx context.Context, vote *domain.Vote) error {
	event := struct {
		Type string       `json:"type"`
		Data *domain.Vote `json:"data"`
	}{
		Type: "poll.voted",
		Data: vote,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal poll voted event: %w", err)
	}

	if err := p.client.Publish(ctx, "events", data).Err(); err != nil {
		return fmt.Errorf("publish poll voted event: %w", err)
	}

	p.logger.Info("published poll voted event",
		zap.String("poll_id", vote.PollID.String()),
		zap.String("user_id", vote.UserID.String()),
		zap.String("option_id", vote.OptionID.String()),
	)

	return nil
}

func (p *RedisPublisher) PublishPollVoteUpdated(ctx context.Context, vote *domain.Vote) error {
	event := struct {
		Type string       `json:"type"`
		Data *domain.Vote `json:"data"`
	}{
		Type: "poll.vote.updated",
		Data: vote,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal poll vote updated event: %w", err)
	}

	if err := p.client.Publish(ctx, "events", data).Err(); err != nil {
		return fmt.Errorf("publish poll vote updated event: %w", err)
	}

	p.logger.Info("published poll vote updated event",
		zap.String("poll_id", vote.PollID.String()),
		zap.String("user_id", vote.UserID.String()),
		zap.String("option_id", vote.OptionID.String()),
	)

	return nil
}

func (p *RedisPublisher) PublishPollVoteDeleted(ctx context.Context, vote *domain.Vote) error {
	event := struct {
		Type string       `json:"type"`
		Data *domain.Vote `json:"data"`
	}{
		Type: "poll.vote.deleted",
		Data: vote,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal poll vote deleted event: %w", err)
	}

	if err := p.client.Publish(ctx, "events", data).Err(); err != nil {
		return fmt.Errorf("publish poll vote deleted event: %w", err)
	}

	p.logger.Info("published poll vote deleted event",
		zap.String("poll_id", vote.PollID.String()),
		zap.String("user_id", vote.UserID.String()),
		zap.String("option_id", vote.OptionID.String()),
	)

	return nil
}

func (p *RedisPublisher) PublishPollSkipped(ctx context.Context, skip *domain.Skip) error {
	event := struct {
		Type string       `json:"type"`
		Data *domain.Skip `json:"data"`
	}{
		Type: "poll.skipped",
		Data: skip,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal poll skipped event: %w", err)
	}

	if err := p.client.Publish(ctx, "events", data).Err(); err != nil {
		return fmt.Errorf("publish poll skipped event: %w", err)
	}

	p.logger.Info("published poll skipped event",
		zap.String("poll_id", skip.PollID.String()),
		zap.String("user_id", skip.UserID.String()),
	)

	return nil
}

func (p *RedisPublisher) Close() error {
	return p.client.Close()
}
