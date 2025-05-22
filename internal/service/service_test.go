package service

import (
	"context"
	"testing"
	"time"

	"github.com/behzadon/vote/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockPublisher struct {
	mock.Mock
}

func (m *MockPublisher) PublishPollCreated(ctx context.Context, poll *domain.Poll) error {
	args := m.Called(ctx, poll)
	return args.Error(0)
}

func (m *MockPublisher) PublishPollVoted(ctx context.Context, vote *domain.Vote) error {
	args := m.Called(ctx, vote)
	return args.Error(0)
}

func (m *MockPublisher) PublishPollSkipped(ctx context.Context, skip *domain.Skip) error {
	args := m.Called(ctx, skip)
	return args.Error(0)
}

func (m *MockPublisher) PublishPollVoteDeleted(ctx context.Context, vote *domain.Vote) error {
	args := m.Called(ctx, vote)
	return args.Error(0)
}

func (m *MockPublisher) PublishPollVoteUpdated(ctx context.Context, vote *domain.Vote) error {
	args := m.Called(ctx, vote)
	return args.Error(0)
}

func (m *MockPublisher) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreatePoll(ctx context.Context, poll *domain.Poll, options []string, tags []string) error {
	args := m.Called(ctx, poll, options, tags)
	return args.Error(0)
}

func (m *MockRepository) GetPollByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Poll), args.Error(1)
}

func (m *MockRepository) GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) ([]domain.Poll, int, error) {
	args := m.Called(ctx, userID, tag, page, limit)
	return args.Get(0).([]domain.Poll), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	args := m.Called(ctx, pollID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PollStats), args.Error(1)
}

func (m *MockRepository) CreateVote(ctx context.Context, pollID, userID, optionID uuid.UUID) error {
	args := m.Called(ctx, pollID, userID, optionID)
	return args.Error(0)
}

func (m *MockRepository) HasVoted(ctx context.Context, pollID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, pollID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	args := m.Called(ctx, userID, date)
	return args.Int(0), args.Error(1)
}

func (m *MockRepository) IncrementUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) error {
	args := m.Called(ctx, userID, date)
	return args.Error(0)
}

func (m *MockRepository) CreateSkip(ctx context.Context, pollID, userID uuid.UUID) error {
	args := m.Called(ctx, pollID, userID)
	return args.Error(0)
}

func (m *MockRepository) HasSkipped(ctx context.Context, pollID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, pollID, userID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetCachedPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	args := m.Called(ctx, pollID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PollStats), args.Error(1)
}

func (m *MockRepository) SetCachedPollStats(ctx context.Context, pollID uuid.UUID, stats *domain.PollStats) error {
	args := m.Called(ctx, pollID, stats)
	return args.Error(0)
}

func (m *MockRepository) InvalidatePollStatsCache(ctx context.Context, pollID uuid.UUID) error {
	args := m.Called(ctx, pollID)
	return args.Error(0)
}

func (m *MockRepository) GetCachedPoll(ctx context.Context, pollID uuid.UUID) (*domain.Poll, error) {
	args := m.Called(ctx, pollID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Poll), args.Error(1)
}

func (m *MockRepository) SetCachedPoll(ctx context.Context, poll *domain.Poll) error {
	args := m.Called(ctx, poll)
	return args.Error(0)
}

func (m *MockRepository) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockRepository) CreateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) DeleteVote(ctx context.Context, voteID, userID uuid.UUID) error {
	args := m.Called(ctx, voteID, userID)
	return args.Error(0)
}

func (m *MockRepository) GetUserVotes(ctx context.Context, userID uuid.UUID, page, limit int) ([]domain.Vote, int, error) {
	args := m.Called(ctx, userID, page, limit)
	return args.Get(0).([]domain.Vote), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetVoteByID(ctx context.Context, voteID uuid.UUID) (*domain.Vote, error) {
	args := m.Called(ctx, voteID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Vote), args.Error(1)
}

func (m *MockRepository) UpdateVote(ctx context.Context, voteID, userID, optionID uuid.UUID) error {
	args := m.Called(ctx, voteID, userID, optionID)
	return args.Error(0)
}

func setupTestService(t *testing.T) (*service, *MockPublisher, *MockRepository) {
	mockPublisher := new(MockPublisher)
	mockRepo := new(MockRepository)
	logger, _ := zap.NewDevelopment()
	svc := &service{
		repo:      mockRepo,
		publisher: mockPublisher,
		logger:    logger,
	}
	return svc, mockPublisher, mockRepo
}

func TestCreatePoll(t *testing.T) {
	tests := []struct {
		name          string
		req           *domain.CreatePollRequest
		setupMocks    func(*MockPublisher, *MockRepository)
		expectedError error
	}{
		{
			name: "successful poll creation",
			req: &domain.CreatePollRequest{
				Title:   "Test Poll",
				Options: []string{"Option 1", "Option 2"},
				Tags:    []string{"test"},
			},
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				repo.On("CreatePoll", mock.Anything, mock.MatchedBy(func(poll *domain.Poll) bool {
					return poll.Title == "Test Poll" &&
						len(poll.Options) == 2 &&
						poll.Options[0].OptionText == "Option 1" &&
						poll.Options[1].OptionText == "Option 2" &&
						poll.Options[0].OptionIndex == 0 &&
						poll.Options[1].OptionIndex == 1 &&
						len(poll.Tags) == 1 &&
						poll.Tags[0] == "test"
				}), []string{"Option 1", "Option 2"}, []string{"test"}).Return(nil)
				pub.On("PublishPollCreated", mock.Anything, mock.MatchedBy(func(poll *domain.Poll) bool {
					return poll.Title == "Test Poll" &&
						len(poll.Options) == 2 &&
						poll.Options[0].OptionText == "Option 1" &&
						poll.Options[1].OptionText == "Option 2" &&
						poll.Options[0].OptionIndex == 0 &&
						poll.Options[1].OptionIndex == 1 &&
						len(poll.Tags) == 1 &&
						poll.Tags[0] == "test"
				})).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "empty title",
			req: &domain.CreatePollRequest{
				Title:   "",
				Options: []string{"Option 1", "Option 2"},
				Tags:    []string{"test"},
			},
			setupMocks:    func(pub *MockPublisher, repo *MockRepository) {},
			expectedError: domain.ErrInvalidInput,
		},
		{
			name: "insufficient options",
			req: &domain.CreatePollRequest{
				Title:   "Test Poll",
				Options: []string{"Option 1"},
				Tags:    []string{"test"},
			},
			setupMocks:    func(pub *MockPublisher, repo *MockRepository) {},
			expectedError: domain.ErrInvalidInput,
		},
		{
			name: "empty tags",
			req: &domain.CreatePollRequest{
				Title:   "Test Poll",
				Options: []string{"Option 1", "Option 2"},
				Tags:    []string{},
			},
			setupMocks:    func(pub *MockPublisher, repo *MockRepository) {},
			expectedError: domain.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, pub, repo := setupTestService(t)
			tt.setupMocks(pub, repo)

			pollID, err := svc.CreatePoll(context.Background(), tt.req)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Equal(t, uuid.Nil, pollID)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, pollID)
			}

			pub.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	}
}

func TestVoteOnPoll(t *testing.T) {
	pollID := uuid.New()
	userID := uuid.New()
	optionID := uuid.New()

	tests := []struct {
		name          string
		pollID        uuid.UUID
		req           *domain.VoteRequest
		setupMocks    func(*MockPublisher, *MockRepository)
		expectedError error
	}{
		{
			name:   "successful vote",
			pollID: pollID,
			req: &domain.VoteRequest{
				UserID:      userID,
				OptionIndex: 0,
			},
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				poll := &domain.Poll{
					ID: pollID,
					Options: []domain.Option{
						{ID: optionID, OptionIndex: 0},
					},
				}
				repo.On("HasVoted", mock.Anything, pollID, userID).Return(false, nil)
				repo.On("GetPollByID", mock.Anything, pollID).Return(poll, nil)
				repo.On("GetUserDailyVoteCount", mock.Anything, userID, mock.Anything).Return(0, nil)
				repo.On("CreateVote", mock.Anything, pollID, userID, optionID).Return(nil)
				repo.On("InvalidatePollStatsCache", mock.Anything, pollID).Return(nil)
				pub.On("PublishPollVoted", mock.Anything, mock.MatchedBy(func(vote *domain.Vote) bool {
					return vote.PollID == pollID && vote.UserID == userID && vote.OptionID == optionID
				})).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "already voted",
			pollID: pollID,
			req: &domain.VoteRequest{
				UserID:      userID,
				OptionIndex: 0,
			},
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				repo.On("HasVoted", mock.Anything, pollID, userID).Return(true, nil)
			},
			expectedError: domain.ErrAlreadyVoted,
		},
		{
			name:   "daily vote limit exceeded",
			pollID: pollID,
			req: &domain.VoteRequest{
				UserID:      userID,
				OptionIndex: 0,
			},
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				poll := &domain.Poll{
					ID: pollID,
					Options: []domain.Option{
						{ID: optionID, OptionIndex: 0},
					},
				}
				repo.On("HasVoted", mock.Anything, pollID, userID).Return(false, nil)
				repo.On("GetPollByID", mock.Anything, pollID).Return(poll, nil)
				repo.On("GetUserDailyVoteCount", mock.Anything, userID, mock.Anything).Return(domain.MaxDailyVotes, nil)
			},
			expectedError: domain.ErrDailyVoteLimitExceeded,
		},
		{
			name:   "invalid option index",
			pollID: pollID,
			req: &domain.VoteRequest{
				UserID:      userID,
				OptionIndex: 1,
			},
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				poll := &domain.Poll{
					ID: pollID,
					Options: []domain.Option{
						{ID: optionID, OptionIndex: 0},
					},
				}
				repo.On("HasVoted", mock.Anything, pollID, userID).Return(false, nil)
				repo.On("GetPollByID", mock.Anything, pollID).Return(poll, nil)
			},
			expectedError: domain.ErrInvalidOption,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, pub, repo := setupTestService(t)
			tt.setupMocks(pub, repo)

			err := svc.VoteOnPoll(context.Background(), tt.pollID, tt.req)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			pub.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	}
}

func TestSkipPoll(t *testing.T) {
	pollID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name          string
		pollID        uuid.UUID
		req           *domain.SkipRequest
		setupMocks    func(*MockPublisher, *MockRepository)
		expectedError error
	}{
		{
			name:   "successful skip",
			pollID: pollID,
			req: &domain.SkipRequest{
				UserID: userID,
			},
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				repo.On("HasSkipped", mock.Anything, pollID, userID).Return(false, nil)
				repo.On("CreateSkip", mock.Anything, pollID, userID).Return(nil)
				pub.On("PublishPollSkipped", mock.Anything, mock.MatchedBy(func(skip *domain.Skip) bool {
					return skip.PollID == pollID && skip.UserID == userID
				})).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:   "already skipped",
			pollID: pollID,
			req: &domain.SkipRequest{
				UserID: userID,
			},
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				repo.On("HasSkipped", mock.Anything, pollID, userID).Return(true, nil)
			},
			expectedError: domain.ErrAlreadySkipped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, pub, repo := setupTestService(t)
			tt.setupMocks(pub, repo)

			err := svc.SkipPoll(context.Background(), tt.pollID, tt.req)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			pub.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	}
}

func TestGetPollStats(t *testing.T) {
	pollID := uuid.New()
	stats := &domain.PollStats{
		PollID: pollID,
		Votes: []domain.OptionStats{
			{Option: "Option 1", Count: 5},
			{Option: "Option 2", Count: 5},
		},
	}

	tests := []struct {
		name          string
		pollID        uuid.UUID
		setupMocks    func(*MockPublisher, *MockRepository)
		expectedStats *domain.PollStats
		expectedError error
	}{
		{
			name:   "get from cache",
			pollID: pollID,
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				repo.On("GetCachedPollStats", mock.Anything, pollID).Return(stats, nil)
			},
			expectedStats: stats,
			expectedError: nil,
		},
		{
			name:   "get from database and cache",
			pollID: pollID,
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				repo.On("GetCachedPollStats", mock.Anything, pollID).Return(nil, domain.ErrNotFound)
				repo.On("GetPollStats", mock.Anything, pollID).Return(stats, nil)
				repo.On("SetCachedPollStats", mock.Anything, pollID, stats).Return(nil)
			},
			expectedStats: stats,
			expectedError: nil,
		},
		{
			name:   "poll not found",
			pollID: pollID,
			setupMocks: func(pub *MockPublisher, repo *MockRepository) {
				repo.On("GetCachedPollStats", mock.Anything, pollID).Return(nil, domain.ErrNotFound)
				repo.On("GetPollStats", mock.Anything, pollID).Return(nil, domain.ErrNotFound)
			},
			expectedStats: nil,
			expectedError: domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, pub, repo := setupTestService(t)
			tt.setupMocks(pub, repo)

			stats, err := svc.GetPollStats(context.Background(), tt.pollID)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, stats)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStats, stats)
			}

			pub.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	}
}
