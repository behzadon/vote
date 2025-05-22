package service

import (
	"context"

	"github.com/behzadon/vote/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockService struct {
	mock.Mock
}

func (m *MockService) CreateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockService) UpdateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockService) CreatePoll(ctx context.Context, req *domain.CreatePollRequest) (uuid.UUID, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockService) GetPollByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Poll), args.Error(1)
}

func (m *MockService) GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) (*domain.PollFeedResponse, error) {
	args := m.Called(ctx, userID, tag, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PollFeedResponse), args.Error(1)
}

func (m *MockService) GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	args := m.Called(ctx, pollID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PollStats), args.Error(1)
}

func (m *MockService) Vote(ctx context.Context, pollID, userID uuid.UUID, optionIndex int) error {
	args := m.Called(ctx, pollID, userID, optionIndex)
	return args.Error(0)
}

func (m *MockService) Skip(ctx context.Context, pollID, userID uuid.UUID) error {
	args := m.Called(ctx, pollID, userID)
	return args.Error(0)
}

func (m *MockService) SkipPoll(ctx context.Context, pollID uuid.UUID, req *domain.SkipRequest) error {
	args := m.Called(ctx, pollID, req)
	return args.Error(0)
}

func (m *MockService) VoteOnPoll(ctx context.Context, pollID uuid.UUID, req *domain.VoteRequest) error {
	args := m.Called(ctx, pollID, req)
	return args.Error(0)
}

func (m *MockService) DeleteVote(ctx context.Context, voteID uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, voteID, userID)
	return args.Error(0)
}

func (m *MockService) GetUserVotes(ctx context.Context, userID uuid.UUID, page, limit int) (*domain.UserVotesResponse, error) {
	args := m.Called(ctx, userID, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UserVotesResponse), args.Error(1)
}

func (m *MockService) UpdateVote(ctx context.Context, voteID uuid.UUID, req *domain.UpdateVoteRequest) error {
	args := m.Called(ctx, voteID, req)
	return args.Error(0)
}
