package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/behzadon/vote/internal/domain"
	"github.com/behzadon/vote/internal/metrics"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func pollKey(id uuid.UUID) string {
	return fmt.Sprintf("poll:%s", id.String())
}

func pollStatsKey(id uuid.UUID) string {
	return fmt.Sprintf("poll:stats:%s", id.String())
}

func userDailyVotesKey(userID uuid.UUID, date time.Time) string {
	return fmt.Sprintf("user:daily:votes:%s:%s", userID.String(), date.Format("2006-01-02"))
}

func (c *RedisCache) GetPoll(ctx context.Context, pollID uuid.UUID) (*domain.Poll, error) {
	key := fmt.Sprintf("poll:%s", pollID.String())
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			metrics.RecordCacheOperation("get_poll", false)
			return nil, nil
		}
		return nil, fmt.Errorf("get poll from cache: %w", err)
	}

	var poll domain.Poll
	if err := json.Unmarshal(data, &poll); err != nil {
		return nil, fmt.Errorf("unmarshal poll: %w", err)
	}

	metrics.RecordCacheOperation("get_poll", true)
	return &poll, nil
}

func (c *RedisCache) SetPoll(ctx context.Context, poll *domain.Poll) error {
	key := fmt.Sprintf("poll:%s", poll.ID.String())
	data, err := json.Marshal(poll)
	if err != nil {
		return fmt.Errorf("marshal poll: %w", err)
	}

	if err := c.client.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("set poll in cache: %w", err)
	}

	metrics.RecordCacheOperation("set_poll", true)
	return nil
}

func (c *RedisCache) DeletePoll(ctx context.Context, id uuid.UUID) error {
	return c.client.Del(ctx, pollKey(id)).Err()
}

func (c *RedisCache) GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	key := fmt.Sprintf("poll_stats:%s", pollID.String())
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			metrics.RecordCacheOperation("get_poll_stats", false)
			return nil, nil
		}
		return nil, fmt.Errorf("get poll stats from cache: %w", err)
	}

	var stats domain.PollStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("unmarshal poll stats: %w", err)
	}

	metrics.RecordCacheOperation("get_poll_stats", true)
	return &stats, nil
}

func (c *RedisCache) SetPollStats(ctx context.Context, pollID uuid.UUID, stats *domain.PollStats) error {
	key := fmt.Sprintf("poll_stats:%s", pollID.String())
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal poll stats: %w", err)
	}

	if err := c.client.Set(ctx, key, data, 1*time.Hour).Err(); err != nil {
		return fmt.Errorf("set poll stats in cache: %w", err)
	}

	metrics.RecordCacheOperation("set_poll_stats", true)
	return nil
}

func (c *RedisCache) DeletePollStats(ctx context.Context, pollID uuid.UUID) error {
	return c.client.Del(ctx, pollStatsKey(pollID)).Err()
}

func (c *RedisCache) GetUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	count, err := c.client.Get(ctx, userDailyVotesKey(userID, date)).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}
	return count, nil
}

func (c *RedisCache) SetUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time, count int) error {
	return c.client.Set(ctx, userDailyVotesKey(userID, date), count, 24*time.Hour).Err()
}

func (c *RedisCache) IncrementUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) error {
	key := userDailyVotesKey(userID, date)
	count, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		return err
	}

	if count > int64(domain.MaxDailyVotes) {
		c.client.Decr(ctx, key)
		return domain.ErrDailyVoteLimitExceeded
	}

	if count == 1 {
		c.client.Expire(ctx, key, 24*time.Hour)
	}

	return nil
}
