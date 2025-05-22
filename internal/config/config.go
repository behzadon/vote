package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Postgres  PostgresConfig  `mapstructure:"postgres"`
	Redis     RedisConfig     `mapstructure:"redis"`
	RabbitMQ  RabbitMQConfig  `mapstructure:"rabbitmq"`
	Migration MigrationConfig `mapstructure:"migration"`
	JWT       JWTConfig       `mapstructure:"jwt"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type RabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
}

type MigrationConfig struct {
	AutoMigrate bool `mapstructure:"auto_migrate"`
}

type JWTConfig struct {
	SecretKey     string        `mapstructure:"secret_key"`
	TokenDuration time.Duration `mapstructure:"token_duration"`
}

func Load(configFile string) (*Config, error) {
	v := viper.New()

	v.SetDefault("server.port", 8080)
	v.SetDefault("server.env", "development")
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.sslmode", "disable")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("rabbitmq.port", 5672)
	v.SetDefault("rabbitmq.vhost", "/")
	v.SetDefault("migration.auto_migrate", false)
	v.SetDefault("jwt.token_duration", 24*time.Hour)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if configFile != "" {
		v.SetConfigFile(configFile)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	if err := bindEnvs(v); err != nil {
		return nil, fmt.Errorf("bind env vars: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func bindEnvs(v *viper.Viper) error {
	bindings := map[string]string{
		"server.port":            "VOTE_SERVER_PORT",
		"server.env":             "VOTE_SERVER_ENV",
		"postgres.host":          "VOTE_POSTGRES_HOST",
		"postgres.port":          "VOTE_POSTGRES_PORT",
		"postgres.user":          "VOTE_POSTGRES_USER",
		"postgres.password":      "VOTE_POSTGRES_PASSWORD",
		"postgres.dbname":        "VOTE_POSTGRES_DBNAME",
		"postgres.sslmode":       "VOTE_POSTGRES_SSLMODE",
		"redis.host":             "VOTE_REDIS_HOST",
		"redis.port":             "VOTE_REDIS_PORT",
		"redis.password":         "VOTE_REDIS_PASSWORD",
		"redis.db":               "VOTE_REDIS_DB",
		"rabbitmq.host":          "VOTE_RABBITMQ_HOST",
		"rabbitmq.port":          "VOTE_RABBITMQ_PORT",
		"rabbitmq.user":          "VOTE_RABBITMQ_USER",
		"rabbitmq.password":      "VOTE_RABBITMQ_PASSWORD",
		"rabbitmq.vhost":         "VOTE_RABBITMQ_VHOST",
		"migration.auto_migrate": "VOTE_MIGRATION_AUTO_MIGRATE",
		"jwt.secret_key":         "VOTE_JWT_SECRET_KEY",
		"jwt.token_duration":     "VOTE_JWT_TOKEN_DURATION",
	}

	for key, env := range bindings {
		if err := v.BindEnv(key, env); err != nil {
			return fmt.Errorf("bind env %s: %w", env, err)
		}
	}

	return nil
}

func validateConfig(cfg *Config) error {
	if cfg.Server.Port <= 0 {
		return fmt.Errorf("server.port must be greater than 0")
	}
	if cfg.Server.Env == "" {
		return fmt.Errorf("server.env is required")
	}

	if cfg.Postgres.Host == "" {
		return fmt.Errorf("postgres.host is required")
	}
	if cfg.Postgres.Port <= 0 {
		return fmt.Errorf("postgres.port must be greater than 0")
	}
	if cfg.Postgres.User == "" {
		return fmt.Errorf("postgres.user is required")
	}
	if cfg.Postgres.DBName == "" {
		return fmt.Errorf("postgres.dbname is required")
	}

	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if cfg.Redis.Port <= 0 {
		return fmt.Errorf("redis.port must be greater than 0")
	}

	if cfg.RabbitMQ.Host == "" {
		return fmt.Errorf("rabbitmq.host is required")
	}
	if cfg.RabbitMQ.Port <= 0 {
		return fmt.Errorf("rabbitmq.port must be greater than 0")
	}
	if cfg.RabbitMQ.User == "" {
		return fmt.Errorf("rabbitmq.user is required")
	}

	if cfg.JWT.SecretKey == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if cfg.JWT.TokenDuration <= 0 {
		return fmt.Errorf("jwt.token_duration must be greater than 0")
	}

	return nil
}
