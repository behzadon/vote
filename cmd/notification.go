package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/behzadon/vote/internal/logging"
	"github.com/behzadon/vote/internal/notification"
	"github.com/behzadon/vote/internal/storage/events"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var notificationConsumerCmd = &cobra.Command{
	Use:   "notification-consumer",
	Short: "Start the notification consumer",
	Long:  `Start the notification consumer that processes poll events and sends notifications.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cfg := GetConfig()

		zapLogger, err := zap.NewProduction()
		if err != nil {
			return fmt.Errorf("create logger: %w", err)
		}
		defer func() {
			if err := zapLogger.Sync(); err != nil {
				zapLogger.Error("Failed to sync logger", zap.Error(err))
			}
		}()

		logger := logging.NewLogger(zapLogger)

		// TODO: Implement a real notification service
		mockNotificationService := &notification.MockNotificationService{
			Logger: zapLogger,
		}

		handler := notification.NewNotificationHandler(mockNotificationService, zapLogger)

		consumer, err := events.NewRabbitMQConsumer(
			cfg.RabbitMQ.Host,
			cfg.RabbitMQ.Port,
			cfg.RabbitMQ.User,
			cfg.RabbitMQ.Password,
			cfg.RabbitMQ.VHost,
			"vote_events",
			handler,
			zapLogger,
		)
		if err != nil {
			return fmt.Errorf("create RabbitMQ consumer: %w", err)
		}
		defer func() {
			if err := consumer.Close(); err != nil {
				logger.Error("Failed to close RabbitMQ consumer", err)
			}
		}()

		if err := consumer.Start(ctx); err != nil {
			return fmt.Errorf("start consumer: %w", err)
		}

		logger.Info("Notification consumer started")

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		logger.Info("Shutting down notification consumer...")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(notificationConsumerCmd)
}
