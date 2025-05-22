package domain

import "errors"

type RepositoryError struct {
	Op  string
	Err error
}

func (e *RepositoryError) Error() string {
	if e.Err == nil {
		return e.Op
	}
	return e.Op + ": " + e.Err.Error()
}

var (
	ErrNotFound               = errors.New("resource not found")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrInvalidInput           = errors.New("invalid input")
	ErrAlreadyVoted           = errors.New("user has already voted on this poll")
	ErrAlreadySkipped         = errors.New("user has already skipped this poll")
	ErrInvalidOption          = errors.New("invalid option index")
	ErrDailyVoteLimitExceeded = errors.New("daily vote limit exceeded")
	ErrInvalidUser            = errors.New("invalid user ID")
	ErrInvalidPoll            = errors.New("invalid poll ID")
	ErrInvalidTag             = errors.New("invalid tag")
	ErrInvalidPageSize        = errors.New("invalid page size")
	ErrEmailAlreadyExists     = errors.New("email already exists")
	ErrUnauthorized           = errors.New("unauthorized")
)
