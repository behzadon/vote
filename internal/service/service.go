package service

import (
	"context"
	"fmt"
	"time"

	"github.com/behzadon/vote/internal/domain"
	"github.com/behzadon/vote/internal/events"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service interface {
	CreatePoll(ctx context.Context, req *domain.CreatePollRequest) (uuid.UUID, error)
	GetPollByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error)
	GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) (*domain.PollFeedResponse, error)
	GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error)

	VoteOnPoll(ctx context.Context, pollID uuid.UUID, req *domain.VoteRequest) error
	UpdateVote(ctx context.Context, voteID uuid.UUID, req *domain.UpdateVoteRequest) error
	DeleteVote(ctx context.Context, voteID uuid.UUID, userID uuid.UUID) error
	SkipPoll(ctx context.Context, pollID uuid.UUID, req *domain.SkipRequest) error
	GetUserVotes(ctx context.Context, userID uuid.UUID, page, limit int) (*domain.UserVotesResponse, error)

	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo      domain.Repository
	publisher events.Publisher
	logger    *zap.Logger
}

func NewService(repo domain.Repository, publisher events.Publisher, logger *zap.Logger) Service {
	return &service{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

func (s *service) CreatePoll(ctx context.Context, req *domain.CreatePollRequest) (uuid.UUID, error) {
	if req == nil {
		return uuid.Nil, domain.ErrInvalidInput
	}

	if req.Title == "" {
		return uuid.Nil, domain.ErrInvalidInput
	}

	if len(req.Options) < 2 {
		return uuid.Nil, domain.ErrInvalidInput
	}

	if len(req.Tags) == 0 {
		return uuid.Nil, domain.ErrInvalidInput
	}

	poll := &domain.Poll{
		ID:        uuid.New(),
		Title:     req.Title,
		Options:   make([]domain.Option, len(req.Options)),
		Tags:      req.Tags,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	for i, opt := range req.Options {
		poll.Options[i] = domain.Option{
			ID:          uuid.New(),
			PollID:      poll.ID,
			OptionText:  opt,
			OptionIndex: i,
			CreatedAt:   time.Now().UTC(),
		}
	}

	err := s.repo.CreatePoll(ctx, poll, req.Options, req.Tags)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create poll: %w", err)
	}

	if err := s.publisher.PublishPollCreated(ctx, poll); err != nil {
		s.logger.Error("failed to publish poll created event",
			zap.Error(err),
			zap.String("poll_id", poll.ID.String()),
		)
	}

	return poll.ID, nil
}

func (s *service) GetPollByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	return s.repo.GetPollByID(ctx, id)
}

func (s *service) GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) (*domain.PollFeedResponse, error) {
	polls, total, err := s.repo.GetPollsForFeed(ctx, userID, tag, page, limit)
	if err != nil {
		return nil, err
	}

	return &domain.PollFeedResponse{
		Polls: polls,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *service) GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	stats, err := s.repo.GetCachedPollStats(ctx, pollID)
	if err == nil {
		return stats, nil
	}

	stats, err = s.repo.GetPollStats(ctx, pollID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SetCachedPollStats(ctx, pollID, stats); err != nil {
		s.logger.Warn("Failed to cache poll stats",
			zap.String("poll_id", pollID.String()),
			zap.Error(err),
		)
	}

	return stats, nil
}

func (s *service) VoteOnPoll(ctx context.Context, pollID uuid.UUID, req *domain.VoteRequest) error {
	hasVoted, err := s.repo.HasVoted(ctx, pollID, req.UserID)
	if err != nil {
		return err
	}
	if hasVoted {
		return domain.ErrAlreadyVoted
	}

	poll, err := s.repo.GetPollByID(ctx, pollID)
	if err != nil {
		return err
	}

	if req.OptionIndex < 0 || req.OptionIndex >= len(poll.Options) {
		return domain.ErrInvalidOption
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	voteCount, err := s.repo.GetUserDailyVoteCount(ctx, req.UserID, today)
	if err != nil {
		return err
	}
	if voteCount >= domain.MaxDailyVotes {
		return domain.ErrDailyVoteLimitExceeded
	}

	vote := &domain.Vote{
		ID:        uuid.New(),
		PollID:    pollID,
		UserID:    req.UserID,
		OptionID:  poll.Options[req.OptionIndex].ID,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.CreateVote(ctx, pollID, req.UserID, poll.Options[req.OptionIndex].ID); err != nil {
		return err
	}

	if err := s.repo.InvalidatePollStatsCache(ctx, pollID); err != nil {
		s.logger.Warn("Failed to invalidate poll stats cache",
			zap.Error(err),
			zap.String("poll_id", pollID.String()),
		)
	}

	if err := s.publisher.PublishPollVoted(ctx, vote); err != nil {
		s.logger.Error("Failed to publish poll voted event",
			zap.Error(err),
			zap.String("poll_id", pollID.String()),
			zap.String("user_id", req.UserID.String()),
		)
	}

	return nil
}

func (s *service) UpdateVote(ctx context.Context, voteID uuid.UUID, req *domain.UpdateVoteRequest) error {
	if req == nil {
		return domain.ErrInvalidInput
	}

	vote, err := s.repo.GetVoteByID(ctx, voteID)
	if err != nil {
		return err
	}

	if vote.UserID != req.UserID {
		return domain.ErrUnauthorized
	}

	poll, err := s.repo.GetPollByID(ctx, vote.PollID)
	if err != nil {
		return err
	}

	if req.OptionIndex < 0 || req.OptionIndex >= len(poll.Options) {
		return domain.ErrInvalidOption
	}

	err = s.repo.UpdateVote(ctx, voteID, req.UserID, poll.Options[req.OptionIndex].ID)
	if err != nil {
		return err
	}

	updatedVote := &domain.Vote{
		ID:        voteID,
		PollID:    vote.PollID,
		UserID:    req.UserID,
		OptionID:  poll.Options[req.OptionIndex].ID,
		CreatedAt: vote.CreatedAt,
	}

	if err := s.publisher.PublishPollVoteUpdated(ctx, updatedVote); err != nil {
		s.logger.Error("Failed to publish poll vote updated event",
			zap.Error(err),
			zap.String("vote_id", voteID.String()),
			zap.String("user_id", req.UserID.String()),
		)
	}

	return nil
}

func (s *service) DeleteVote(ctx context.Context, voteID, userID uuid.UUID) error {
	vote, err := s.repo.GetVoteByID(ctx, voteID)
	if err != nil {
		return err
	}

	if vote.UserID != userID {
		return domain.ErrUnauthorized
	}

	err = s.repo.DeleteVote(ctx, voteID, userID)
	if err != nil {
		return err
	}

	if err := s.publisher.PublishPollVoteDeleted(ctx, vote); err != nil {
		s.logger.Error("Failed to publish poll vote deleted event",
			zap.Error(err),
			zap.String("vote_id", voteID.String()),
			zap.String("user_id", userID.String()),
		)
	}

	return nil
}

func (s *service) SkipPoll(ctx context.Context, pollID uuid.UUID, req *domain.SkipRequest) error {
	hasSkipped, err := s.repo.HasSkipped(ctx, pollID, req.UserID)
	if err != nil {
		return err
	}
	if hasSkipped {
		return domain.ErrAlreadySkipped
	}

	skip := &domain.Skip{
		ID:        uuid.New(),
		PollID:    pollID,
		UserID:    req.UserID,
		CreatedAt: time.Now().UTC(),
	}

	err = s.repo.CreateSkip(ctx, pollID, req.UserID)
	if err != nil {
		return err
	}

	if err := s.publisher.PublishPollSkipped(ctx, skip); err != nil {
		s.logger.Error("Failed to publish poll skipped event",
			zap.Error(err),
			zap.String("poll_id", pollID.String()),
			zap.String("user_id", req.UserID.String()),
		)
	}

	return nil
}

func (s *service) GetUserVotes(ctx context.Context, userID uuid.UUID, page, limit int) (*domain.UserVotesResponse, error) {
	if page < 1 {
		page = domain.DefaultPage
	}
	if limit < 1 || limit > domain.MaxPageSize {
		limit = domain.DefaultLimit
	}

	votes, total, err := s.repo.GetUserVotes(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}

	voteResponses := make([]domain.VoteResponse, len(votes))
	for i, vote := range votes {
		voteResponses[i] = domain.VoteResponse{
			ID:         vote.ID,
			PollID:     vote.PollID,
			OptionID:   vote.OptionID,
			CreatedAt:  vote.CreatedAt,
			PollTitle:  vote.PollTitle,
			OptionText: vote.OptionText,
		}
	}

	return &domain.UserVotesResponse{
		Votes: voteResponses,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (s *service) CreateUser(ctx context.Context, user *domain.User) error {
	return s.repo.CreateUser(ctx, user)
}

func (s *service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.repo.GetUserByEmail(ctx, email)
}

func (s *service) UpdateUser(ctx context.Context, user *domain.User) error {
	return s.repo.UpdateUser(ctx, user)
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteUser(ctx, id)
}
