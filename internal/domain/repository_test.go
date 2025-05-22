package domain

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreatePoll(ctx context.Context, poll *Poll, options []string, tags []string) error {
	args := m.Called(ctx, poll, options, tags)
	return args.Error(0)
}

func (m *MockRepository) GetPollByID(ctx context.Context, id uuid.UUID) (*Poll, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Poll), args.Error(1)
}

func (m *MockRepository) GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) ([]Poll, int, error) {
	args := m.Called(ctx, userID, tag, page, limit)
	return args.Get(0).([]Poll), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetPollStats(ctx context.Context, pollID uuid.UUID) (*PollStats, error) {
	args := m.Called(ctx, pollID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PollStats), args.Error(1)
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

func (m *MockRepository) GetCachedPollStats(ctx context.Context, pollID uuid.UUID) (*PollStats, error) {
	args := m.Called(ctx, pollID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PollStats), args.Error(1)
}

func (m *MockRepository) SetCachedPollStats(ctx context.Context, pollID uuid.UUID, stats *PollStats) error {
	args := m.Called(ctx, pollID, stats)
	return args.Error(0)
}

func (m *MockRepository) InvalidatePollStatsCache(ctx context.Context, pollID uuid.UUID) error {
	args := m.Called(ctx, pollID)
	return args.Error(0)
}

func (m *MockRepository) GetCachedPoll(ctx context.Context, id uuid.UUID) (*Poll, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Poll), args.Error(1)
}

func (m *MockRepository) SetCachedPoll(ctx context.Context, poll *Poll) error {
	args := m.Called(ctx, poll)
	return args.Error(0)
}

func (m *MockRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func TestMockRepository_CreatePoll(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	poll := &Poll{ID: uuid.New(), Title: "Test Poll", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	options := []string{"Option 1", "Option 2"}
	tags := []string{"test"}

	mockRepo.On("CreatePoll", ctx, poll, options, tags).Return(nil).Once()

	err := mockRepo.CreatePoll(ctx, poll, options, tags)
	assert.NoError(t, err, "CreatePoll should not return an error")
	mockRepo.AssertExpectations(t)

	mockRepo.On("CreatePoll", ctx, poll, options, tags).Return(ErrInvalidInput).Once()
	err = mockRepo.CreatePoll(ctx, poll, options, tags)
	assert.Equal(t, ErrInvalidInput, err, "CreatePoll should return ErrInvalidInput")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetPollByID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID := uuid.New()
	poll := &Poll{ID: pollID, Title: "Test Poll", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	mockRepo.On("GetPollByID", ctx, pollID).Return(poll, nil).Once()

	got, err := mockRepo.GetPollByID(ctx, pollID)
	assert.NoError(t, err, "GetPollByID should not return an error")
	assert.Equal(t, poll, got, "GetPollByID should return the expected poll")
	mockRepo.AssertExpectations(t)

	mockRepo.On("GetPollByID", ctx, pollID).Return(nil, ErrNotFound).Once()
	got, err = mockRepo.GetPollByID(ctx, pollID)
	assert.Equal(t, ErrNotFound, err, "GetPollByID should return ErrNotFound")
	assert.Nil(t, got, "GetPollByID should return nil poll on error")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetPollsForFeed(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	userID := uuid.New()
	tag := "test"
	page, limit := 1, 10
	polls := []Poll{{ID: uuid.New(), Title: "Feed Poll", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	total := 1

	mockRepo.On("GetPollsForFeed", ctx, userID, tag, page, limit).Return(polls, total, nil).Once()

	gotPolls, gotTotal, err := mockRepo.GetPollsForFeed(ctx, userID, tag, page, limit)
	assert.NoError(t, err, "GetPollsForFeed should not return an error")
	assert.Equal(t, polls, gotPolls, "GetPollsForFeed should return the expected polls")
	assert.Equal(t, total, gotTotal, "GetPollsForFeed should return the expected total")
	mockRepo.AssertExpectations(t)

	mockRepo.On("GetPollsForFeed", ctx, userID, tag, page, limit).Return([]Poll(nil), 0, ErrInvalidInput).Once()
	gotPolls, gotTotal, err = mockRepo.GetPollsForFeed(ctx, userID, tag, page, limit)
	assert.Equal(t, ErrInvalidInput, err, "GetPollsForFeed should return ErrInvalidInput")
	assert.Nil(t, gotPolls, "GetPollsForFeed should return nil polls on error")
	assert.Equal(t, 0, gotTotal, "GetPollsForFeed should return 0 total on error")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetPollStats(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID := uuid.New()
	stats := &PollStats{PollID: pollID, Votes: []OptionStats{{Option: "Option 1", Count: 5}}}

	mockRepo.On("GetPollStats", ctx, pollID).Return(stats, nil).Once()

	got, err := mockRepo.GetPollStats(ctx, pollID)
	assert.NoError(t, err, "GetPollStats should not return an error")
	assert.Equal(t, stats, got, "GetPollStats should return the expected stats")
	mockRepo.AssertExpectations(t)

	mockRepo.On("GetPollStats", ctx, pollID).Return(nil, ErrNotFound).Once()
	got, err = mockRepo.GetPollStats(ctx, pollID)
	assert.Equal(t, ErrNotFound, err, "GetPollStats should return ErrNotFound")
	assert.Nil(t, got, "GetPollStats should return nil stats on error")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_CreateVote(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID, userID, optionID := uuid.New(), uuid.New(), uuid.New()

	mockRepo.On("CreateVote", ctx, pollID, userID, optionID).Return(nil).Once()

	err := mockRepo.CreateVote(ctx, pollID, userID, optionID)
	assert.NoError(t, err, "CreateVote should not return an error")
	mockRepo.AssertExpectations(t)

	mockRepo.On("CreateVote", ctx, pollID, userID, optionID).Return(ErrAlreadyVoted).Once()
	err = mockRepo.CreateVote(ctx, pollID, userID, optionID)
	assert.Equal(t, ErrAlreadyVoted, err, "CreateVote should return ErrAlreadyVoted")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_HasVoted(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID, userID := uuid.New(), uuid.New()

	mockRepo.On("HasVoted", ctx, pollID, userID).Return(true, nil).Once()

	hasVoted, err := mockRepo.HasVoted(ctx, pollID, userID)
	assert.NoError(t, err, "HasVoted should not return an error")
	assert.True(t, hasVoted, "HasVoted should return true")
	mockRepo.AssertExpectations(t)

	mockRepo.On("HasVoted", ctx, pollID, userID).Return(false, ErrNotFound).Once()
	hasVoted, err = mockRepo.HasVoted(ctx, pollID, userID)
	assert.Equal(t, ErrNotFound, err, "HasVoted should return ErrNotFound")
	assert.False(t, hasVoted, "HasVoted should return false on error")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetUserDailyVoteCount(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	userID := uuid.New()
	date := time.Now().UTC().Truncate(24 * time.Hour)

	mockRepo.On("GetUserDailyVoteCount", ctx, userID, date).Return(5, nil).Once()

	count, err := mockRepo.GetUserDailyVoteCount(ctx, userID, date)
	assert.NoError(t, err, "GetUserDailyVoteCount should not return an error")
	assert.Equal(t, 5, count, "GetUserDailyVoteCount should return the expected count")
	mockRepo.AssertExpectations(t)

	mockRepo.On("GetUserDailyVoteCount", ctx, userID, date).Return(0, ErrInvalidUser).Once()
	count, err = mockRepo.GetUserDailyVoteCount(ctx, userID, date)
	assert.Equal(t, ErrInvalidUser, err, "GetUserDailyVoteCount should return ErrInvalidUser")
	assert.Equal(t, 0, count, "GetUserDailyVoteCount should return 0 on error")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_IncrementUserDailyVoteCount(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	userID := uuid.New()
	date := time.Now().UTC().Truncate(24 * time.Hour)

	mockRepo.On("IncrementUserDailyVoteCount", ctx, userID, date).Return(nil).Once()

	err := mockRepo.IncrementUserDailyVoteCount(ctx, userID, date)
	assert.NoError(t, err, "IncrementUserDailyVoteCount should not return an error")
	mockRepo.AssertExpectations(t)

	mockRepo.On("IncrementUserDailyVoteCount", ctx, userID, date).Return(ErrDailyVoteLimitExceeded).Once()
	err = mockRepo.IncrementUserDailyVoteCount(ctx, userID, date)
	assert.Equal(t, ErrDailyVoteLimitExceeded, err, "IncrementUserDailyVoteCount should return ErrDailyVoteLimitExceeded")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_CreateSkip(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID, userID := uuid.New(), uuid.New()

	mockRepo.On("CreateSkip", ctx, pollID, userID).Return(nil).Once()

	err := mockRepo.CreateSkip(ctx, pollID, userID)
	assert.NoError(t, err, "CreateSkip should not return an error")
	mockRepo.AssertExpectations(t)

	mockRepo.On("CreateSkip", ctx, pollID, userID).Return(ErrAlreadySkipped).Once()
	err = mockRepo.CreateSkip(ctx, pollID, userID)
	assert.Equal(t, ErrAlreadySkipped, err, "CreateSkip should return ErrAlreadySkipped")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_HasSkipped(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID, userID := uuid.New(), uuid.New()

	mockRepo.On("HasSkipped", ctx, pollID, userID).Return(true, nil).Once()

	hasSkipped, err := mockRepo.HasSkipped(ctx, pollID, userID)
	assert.NoError(t, err, "HasSkipped should not return an error")
	assert.True(t, hasSkipped, "HasSkipped should return true")
	mockRepo.AssertExpectations(t)

	mockRepo.On("HasSkipped", ctx, pollID, userID).Return(false, ErrNotFound).Once()
	hasSkipped, err = mockRepo.HasSkipped(ctx, pollID, userID)
	assert.Equal(t, ErrNotFound, err, "HasSkipped should return ErrNotFound")
	assert.False(t, hasSkipped, "HasSkipped should return false on error")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetCachedPollStats(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID := uuid.New()
	stats := &PollStats{PollID: pollID, Votes: []OptionStats{{Option: "Option 1", Count: 5}}}

	mockRepo.On("GetCachedPollStats", ctx, pollID).Return(stats, nil).Once()

	got, err := mockRepo.GetCachedPollStats(ctx, pollID)
	assert.NoError(t, err, "GetCachedPollStats should not return an error")
	assert.Equal(t, stats, got, "GetCachedPollStats should return the expected stats")
	mockRepo.AssertExpectations(t)

	mockRepo.On("GetCachedPollStats", ctx, pollID).Return(nil, ErrNotFound).Once()
	got, err = mockRepo.GetCachedPollStats(ctx, pollID)
	assert.Equal(t, ErrNotFound, err, "GetCachedPollStats should return ErrNotFound")
	assert.Nil(t, got, "GetCachedPollStats should return nil stats on error")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_SetCachedPollStats(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID := uuid.New()
	stats := &PollStats{PollID: pollID, Votes: []OptionStats{{Option: "Option 1", Count: 5}}}

	mockRepo.On("SetCachedPollStats", ctx, pollID, stats).Return(nil).Once()

	err := mockRepo.SetCachedPollStats(ctx, pollID, stats)
	assert.NoError(t, err, "SetCachedPollStats should not return an error")
	mockRepo.AssertExpectations(t)

	mockRepo.On("SetCachedPollStats", ctx, pollID, stats).Return(ErrInvalidInput).Once()
	err = mockRepo.SetCachedPollStats(ctx, pollID, stats)
	assert.Equal(t, ErrInvalidInput, err, "SetCachedPollStats should return ErrInvalidInput")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_InvalidatePollStatsCache(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID := uuid.New()

	mockRepo.On("InvalidatePollStatsCache", ctx, pollID).Return(nil).Once()

	err := mockRepo.InvalidatePollStatsCache(ctx, pollID)
	assert.NoError(t, err, "InvalidatePollStatsCache should not return an error")
	mockRepo.AssertExpectations(t)

	mockRepo.On("InvalidatePollStatsCache", ctx, pollID).Return(ErrInvalidInput).Once()
	err = mockRepo.InvalidatePollStatsCache(ctx, pollID)
	assert.Equal(t, ErrInvalidInput, err, "InvalidatePollStatsCache should return ErrInvalidInput")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetCachedPoll(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	pollID := uuid.New()
	poll := &Poll{ID: pollID, Title: "Cached Poll", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	mockRepo.On("GetCachedPoll", ctx, pollID).Return(poll, nil).Once()

	got, err := mockRepo.GetCachedPoll(ctx, pollID)
	assert.NoError(t, err, "GetCachedPoll should not return an error")
	assert.Equal(t, poll, got, "GetCachedPoll should return the expected poll")
	mockRepo.AssertExpectations(t)

	mockRepo.On("GetCachedPoll", ctx, pollID).Return(nil, ErrNotFound).Once()
	got, err = mockRepo.GetCachedPoll(ctx, pollID)
	assert.Equal(t, ErrNotFound, err, "GetCachedPoll should return ErrNotFound")
	assert.Nil(t, got, "GetCachedPoll should return nil poll on error")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_SetCachedPoll(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	poll := &Poll{ID: uuid.New(), Title: "Cached Poll", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	mockRepo.On("SetCachedPoll", ctx, poll).Return(nil).Once()

	err := mockRepo.SetCachedPoll(ctx, poll)
	assert.NoError(t, err, "SetCachedPoll should not return an error")
	mockRepo.AssertExpectations(t)

	mockRepo.On("SetCachedPoll", ctx, poll).Return(ErrInvalidInput).Once()
	err = mockRepo.SetCachedPoll(ctx, poll)
	assert.Equal(t, ErrInvalidInput, err, "SetCachedPoll should return ErrInvalidInput")
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_WithTransaction(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	fn := func(ctx context.Context) error { return nil }

	mockRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil).Once()

	err := mockRepo.WithTransaction(ctx, fn)
	assert.NoError(t, err, "WithTransaction should not return an error")
	mockRepo.AssertExpectations(t)

	mockRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(ErrInvalidInput).Once()
	err = mockRepo.WithTransaction(ctx, fn)
	assert.Equal(t, ErrInvalidInput, err, "WithTransaction should return ErrInvalidInput")
	mockRepo.AssertExpectations(t)
}
