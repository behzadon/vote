package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreatePoll(ctx context.Context, poll *Poll, options []string, tags []string) error
	GetPollByID(ctx context.Context, id uuid.UUID) (*Poll, error)
	GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) ([]Poll, int, error)
	GetPollStats(ctx context.Context, pollID uuid.UUID) (*PollStats, error)

	CreateVote(ctx context.Context, pollID, userID, optionID uuid.UUID) error
	UpdateVote(ctx context.Context, voteID, userID, optionID uuid.UUID) error
	DeleteVote(ctx context.Context, voteID, userID uuid.UUID) error
	HasVoted(ctx context.Context, pollID, userID uuid.UUID) (bool, error)
	GetUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error)
	IncrementUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) error
	GetUserVotes(ctx context.Context, userID uuid.UUID, page, limit int) ([]Vote, int, error)
	GetVoteByID(ctx context.Context, voteID uuid.UUID) (*Vote, error)

	CreateSkip(ctx context.Context, pollID, userID uuid.UUID) error
	HasSkipped(ctx context.Context, pollID, userID uuid.UUID) (bool, error)

	GetCachedPollStats(ctx context.Context, pollID uuid.UUID) (*PollStats, error)
	SetCachedPollStats(ctx context.Context, pollID uuid.UUID, stats *PollStats) error
	InvalidatePollStatsCache(ctx context.Context, pollID uuid.UUID) error

	GetCachedPoll(ctx context.Context, id uuid.UUID) (*Poll, error)
	SetCachedPoll(ctx context.Context, poll *Poll) error

	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
}
