package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/behzadon/vote/internal/api"
	"github.com/behzadon/vote/internal/auth"
	"github.com/behzadon/vote/internal/config"
	"github.com/behzadon/vote/internal/logging"
	"github.com/behzadon/vote/internal/service"
	"github.com/behzadon/vote/internal/storage/events"
	"github.com/behzadon/vote/internal/storage/postgres"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the vote server",
	Long:  `Start the vote server with the specified configuration.`,
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

		db, err := connectPostgres(cfg.Postgres)
		if err != nil {
			return fmt.Errorf("connect to postgres: %w", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				logger.Error("Failed to close database connection", err)
			}
		}()

		if cfg.Migration.AutoMigrate {
			logger.Info("Auto-migration is enabled, running migrations...")
			if err := runMigrations("up"); err != nil {
				return fmt.Errorf("run migrations: %w", err)
			}
			logger.Info("Migrations completed successfully")
		} else {
			logger.Info("Auto-migration is disabled, skipping migrations")
		}

		redisClient, err := connectRedis(cfg.Redis)
		if err != nil {
			return fmt.Errorf("connect to redis: %w", err)
		}
		defer func() {
			if err := redisClient.Close(); err != nil {
				logger.Error("Failed to close Redis connection", err)
			}
		}()

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := redisClient.Ping(pingCtx).Err(); err != nil {
			logger.Error("Failed to connect to Redis", err)
			return fmt.Errorf("redis ping: %w", err)
		}
		logger.Info("Successfully connected to Redis")

		publisher, err := events.NewRabbitMQPublisher(
			cfg.RabbitMQ.Host,
			cfg.RabbitMQ.Port,
			cfg.RabbitMQ.User,
			cfg.RabbitMQ.Password,
			cfg.RabbitMQ.VHost,
			zapLogger,
		)
		if err != nil {
			return fmt.Errorf("create RabbitMQ publisher: %w", err)
		}
		defer func() {
			if err := publisher.Close(); err != nil {
				logger.Error("Failed to close RabbitMQ publisher", err)
			}
		}()

		repo := postgres.NewRepository(db, redisClient, zapLogger)
		svc := service.NewService(repo, publisher, zapLogger)

		jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.TokenDuration)
		authHandler := api.NewAuthHandler(svc, jwtManager, zapLogger)
		handler := api.NewHandler(svc, redisClient, zapLogger, authHandler)

		engine := gin.New()
		engine.Use(gin.Recovery())
		engine.Use(logger.GinLogger())
		engine.Use(handler.Middleware())
		handler.RegisterRoutes(engine, jwtManager)

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
			Handler: engine,
		}

		go func() {
			logger.Info("Starting server",
				zap.Int("port", cfg.Server.Port),
			)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatal("Failed to start server", err)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		logger.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server forced to shutdown", err)
			return fmt.Errorf("server shutdown: %w", err)
		}

		logger.Info("Server exited properly")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func connectPostgres(cfg config.PostgresConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func connectRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
