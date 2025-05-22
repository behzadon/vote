package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepositoryError_Error(t *testing.T) {
	t.Run("with inner error", func(t *testing.T) {
		err := &RepositoryError{Op: "GetPoll", Err: errors.New("db error")}
		assert.Equal(t, "GetPoll: db error", err.Error())
	})
	t.Run("without inner error", func(t *testing.T) {
		err := &RepositoryError{Op: "GetPoll"}
		assert.Equal(t, "GetPoll", err.Error())
	})
}

func TestDomainErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"NotFound", ErrNotFound, "resource not found"},
		{"InvalidInput", ErrInvalidInput, "invalid input"},
		{"AlreadyVoted", ErrAlreadyVoted, "user has already voted on this poll"},
		{"AlreadySkipped", ErrAlreadySkipped, "user has already skipped this poll"},
		{"InvalidOption", ErrInvalidOption, "invalid option index"},
		{"DailyVoteLimitExceeded", ErrDailyVoteLimitExceeded, "daily vote limit exceeded"},
		{"InvalidUser", ErrInvalidUser, "invalid user ID"},
		{"InvalidPoll", ErrInvalidPoll, "invalid poll ID"},
		{"InvalidTag", ErrInvalidTag, "invalid tag"},
		{"InvalidPageSize", ErrInvalidPageSize, "invalid page size"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Error message not equal:\nexpected: %q\nactual  : %q", tt.expected, tt.err.Error())
			}
		})
	}
}
