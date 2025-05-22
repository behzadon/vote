package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/behzadon/vote/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	logger  *zap.Logger
}

func cleanup(ch *amqp.Channel, conn *amqp.Connection, logger *zap.Logger) {
	if ch != nil {
		if err := ch.Close(); err != nil {
			logger.Error("Failed to close RabbitMQ channel", zap.Error(err))
		}
	}
	if conn != nil {
		if err := conn.Close(); err != nil {
			logger.Error("Failed to close RabbitMQ connection", zap.Error(err))
		}
	}
}

func NewRabbitMQPublisher(host string, port int, user, password, vhost string, logger *zap.Logger) (*RabbitMQPublisher, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", user, password, host, port, vhost)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		cleanup(nil, conn, logger)
		return nil, fmt.Errorf("open channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		"vote",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		cleanup(ch, conn, logger)
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	queues := []string{"vote_events", "poll_updates"}
	for _, queue := range queues {
		_, err = ch.QueueDeclare(
			queue,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			cleanup(ch, conn, logger)
			return nil, fmt.Errorf("declare queue %s: %w", queue, err)
		}

		err = ch.QueueBind(
			queue,
			"poll.*",
			"vote",
			false,
			nil,
		)
		if err != nil {
			cleanup(ch, conn, logger)
			return nil, fmt.Errorf("bind queue %s: %w", queue, err)
		}
	}

	return &RabbitMQPublisher{
		conn:    conn,
		channel: ch,
		logger:  logger,
	}, nil
}

func (p *RabbitMQPublisher) Close() error {
	var errs []error

	if err := p.channel.Close(); err != nil {
		p.logger.Error("Failed to close RabbitMQ channel", zap.Error(err))
		errs = append(errs, fmt.Errorf("close channel: %w", err))
	}

	if err := p.conn.Close(); err != nil {
		p.logger.Error("Failed to close RabbitMQ connection", zap.Error(err))
		errs = append(errs, fmt.Errorf("close connection: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}
	return nil
}

func (p *RabbitMQPublisher) PublishPollCreated(ctx context.Context, poll *domain.Poll) error {
	event := struct {
		Type      string       `json:"type"`
		Timestamp string       `json:"timestamp"`
		Data      *domain.Poll `json:"data"`
	}{
		Type:      "poll.created",
		Timestamp: poll.CreatedAt.Format(time.RFC3339),
		Data:      poll,
	}

	return p.publishEvent(ctx, event, "poll.created")
}

func (p *RabbitMQPublisher) PublishPollVoted(ctx context.Context, vote *domain.Vote) error {
	event := struct {
		Type      string       `json:"type"`
		Timestamp string       `json:"timestamp"`
		Data      *domain.Vote `json:"data"`
	}{
		Type:      "poll.voted",
		Timestamp: vote.CreatedAt.Format(time.RFC3339),
		Data:      vote,
	}

	return p.publishEvent(ctx, event, "poll.voted")
}

func (p *RabbitMQPublisher) PublishPollSkipped(ctx context.Context, skip *domain.Skip) error {
	event := struct {
		Type      string       `json:"type"`
		Timestamp string       `json:"timestamp"`
		Data      *domain.Skip `json:"data"`
	}{
		Type:      "poll.skipped",
		Timestamp: skip.CreatedAt.Format(time.RFC3339),
		Data:      skip,
	}

	return p.publishEvent(ctx, event, "poll.skipped")
}

func (p *RabbitMQPublisher) PublishPollVoteDeleted(ctx context.Context, vote *domain.Vote) error {
	event := struct {
		Type      string       `json:"type"`
		Timestamp string       `json:"timestamp"`
		Data      *domain.Vote `json:"data"`
	}{
		Type:      "poll.vote.deleted",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      vote,
	}
	return p.publishEvent(ctx, event, "poll.vote.deleted")
}

func (p *RabbitMQPublisher) PublishPollVoteUpdated(ctx context.Context, vote *domain.Vote) error {
	event := struct {
		Type      string       `json:"type"`
		Timestamp string       `json:"timestamp"`
		Data      *domain.Vote `json:"data"`
	}{
		Type:      "poll.vote.updated",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      vote,
	}
	return p.publishEvent(ctx, event, "poll.vote.updated")
}

func (p *RabbitMQPublisher) publishEvent(ctx context.Context, event interface{}, routingKey string) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	err = p.channel.PublishWithContext(ctx,
		"vote",
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         data,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		p.logger.Error("Failed to publish message to RabbitMQ",
			zap.Error(err),
			zap.String("event_type", fmt.Sprintf("%T", event)),
			zap.String("routing_key", routingKey),
		)
		return fmt.Errorf("publish message: %w", err)
	}

	return nil
}
