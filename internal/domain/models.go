package domain

import (
	"time"

	"github.com/google/uuid"
)

type Poll struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Options   []Option  `json:"options"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Option struct {
	ID          uuid.UUID `json:"id"`
	PollID      uuid.UUID `json:"pollId"`
	OptionText  string    `json:"optionText"`
	OptionIndex int       `json:"optionIndex"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Vote struct {
	ID         uuid.UUID `json:"id"`
	PollID     uuid.UUID `json:"pollId"`
	UserID     uuid.UUID `json:"userId"`
	OptionID   uuid.UUID `json:"optionId"`
	CreatedAt  time.Time `json:"createdAt"`
	PollTitle  string    `json:"pollTitle,omitempty"`
	OptionText string    `json:"optionText,omitempty"`
}

type VoteResponse struct {
	ID         uuid.UUID `json:"id"`
	PollID     uuid.UUID `json:"pollId"`
	OptionID   uuid.UUID `json:"optionId"`
	CreatedAt  time.Time `json:"createdAt"`
	PollTitle  string    `json:"pollTitle,omitempty"`
	OptionText string    `json:"optionText,omitempty"`
}

type Skip struct {
	ID        uuid.UUID `json:"id"`
	PollID    uuid.UUID `json:"pollId"`
	UserID    uuid.UUID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserDailyVotes struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"userId"`
	VoteDate  time.Time `json:"voteDate"`
	VoteCount int       `json:"voteCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type PollStats struct {
	PollID uuid.UUID     `json:"pollId"`
	Votes  []OptionStats `json:"votes"`
}

type OptionStats struct {
	Option string `json:"option"`
	Count  int    `json:"count"`
}

type CreatePollRequest struct {
	Title   string   `json:"title" binding:"required"`
	Options []string `json:"options" binding:"required,min=2"`
	Tags    []string `json:"tags" binding:"required,min=1"`
}

type VoteRequest struct {
	UserID      uuid.UUID `json:"userId" binding:"required"`
	OptionIndex int       `json:"optionIndex" binding:"required,min=0"`
}

type SkipRequest struct {
	UserID uuid.UUID `json:"userId" binding:"required"`
}

type FeedQuery struct {
	Tag    string    `form:"tag"`
	Page   int       `form:"page,default=1" binding:"min=1"`
	Limit  int       `form:"limit,default=10" binding:"min=1,max=100"`
	UserID uuid.UUID `form:"userId" binding:"required"`
}

type PollFeedRequest struct {
	UserID uuid.UUID `json:"userId" binding:"required"`
	Tag    string    `json:"tag"`
	Page   int       `json:"page" binding:"min=1"`
	Limit  int       `json:"limit" binding:"min=1,max=50"`
}

type PollFeedResponse struct {
	Polls []Poll `json:"polls"`
	Total int    `json:"total"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type UserInfo struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserVotesResponse struct {
	Votes []VoteResponse `json:"votes"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

type UpdateVoteRequest struct {
	UserID      uuid.UUID `json:"userId" binding:"required"`
	OptionIndex int       `json:"optionIndex" binding:"required,min=0"`
}

const (
	MaxDailyVotes = 100
	MaxPageSize   = 100
	DefaultPage   = 1
	DefaultLimit  = 10
)
