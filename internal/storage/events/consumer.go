package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/behzadon/vote/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type EventHandler interface {
	HandlePollCreated(ctx context.Context, poll *domain.Poll) error
	HandlePollVoted(ctx context.Context, vote *domain.Vote) error
	HandlePollSkipped(ctx context.Context, skip *domain.Skip) error
}

type RabbitMQConsumer struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	handler   EventHandler
	logger    *zap.Logger
	queueName string
}

func NewRabbitMQConsumer(
	host string,
	port int,
	user, password, vhost string,
	queueName string,
	handler EventHandler,
	logger *zap.Logger,
) (*RabbitMQConsumer, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", user, password, host, port, vhost)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}
	err = ch.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("set QoS: %w", err)
	}

	return &RabbitMQConsumer{
		conn:      conn,
		channel:   ch,
		handler:   handler,
		logger:    logger,
		queueName: queueName,
	}, nil
}

func (c *RabbitMQConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("register consumer: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					c.logger.Error("Consumer channel closed")
					return
				}

				if err := c.handleMessage(ctx, msg); err != nil {
					c.logger.Error("Failed to handle message",
						zap.Error(err),
						zap.String("routing_key", msg.RoutingKey),
					)
					if err := msg.Nack(false, true); err != nil {
						c.logger.Error("Failed to nack message", zap.Error(err))
					}
					continue
				}

				if err := msg.Ack(false); err != nil {
					c.logger.Error("Failed to ack message", zap.Error(err))
				}
			}
		}
	}()

	return nil
}

func (c *RabbitMQConsumer) handleMessage(ctx context.Context, msg amqp.Delivery) error {
	var event struct {
		Type      string          `json:"type"`
		Timestamp string          `json:"timestamp"`
		Data      json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(msg.Body, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	switch event.Type {
	case "poll.created":
		var poll domain.Poll
		if err := json.Unmarshal(event.Data, &poll); err != nil {
			return fmt.Errorf("unmarshal poll: %w", err)
		}
		return c.handler.HandlePollCreated(ctx, &poll)

	case "poll.voted":
		var vote domain.Vote
		if err := json.Unmarshal(event.Data, &vote); err != nil {
			return fmt.Errorf("unmarshal vote: %w", err)
		}
		return c.handler.HandlePollVoted(ctx, &vote)

	case "poll.skipped":
		var skip domain.Skip
		if err := json.Unmarshal(event.Data, &skip); err != nil {
			return fmt.Errorf("unmarshal skip: %w", err)
		}
		return c.handler.HandlePollSkipped(ctx, &skip)

	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
}

func (c *RabbitMQConsumer) Close() error {
	var errs []error

	if err := c.channel.Close(); err != nil {
		c.logger.Error("Failed to close RabbitMQ channel", zap.Error(err))
		errs = append(errs, fmt.Errorf("close channel: %w", err))
	}

	if err := c.conn.Close(); err != nil {
		c.logger.Error("Failed to close RabbitMQ connection", zap.Error(err))
		errs = append(errs, fmt.Errorf("close connection: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}
	return nil
}
