package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/behzadon/vote/internal/domain"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type PostgresRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewPostgresRepository(db *sql.DB, logger *zap.Logger) *PostgresRepository {
	return &PostgresRepository{db: db, logger: logger}
}

func (r *PostgresRepository) CreatePoll(ctx context.Context, poll *domain.Poll, options []string, tags []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx, r.logger)

	query := `INSERT INTO polls (id, title, created_at, updated_at) VALUES ($1, $2, $3, $4)`
	_, err = tx.ExecContext(ctx, query, poll.ID, poll.Title, poll.CreatedAt, poll.UpdatedAt)
	if err != nil {
		return err
	}

	optionsQuery := `INSERT INTO poll_options (id, poll_id, option_text, option_index, created_at) VALUES ($1, $2, $3, $4, $5)`
	for i, optionText := range options {
		optionID := uuid.New()
		_, err = tx.ExecContext(ctx, optionsQuery, optionID, poll.ID, optionText, i, time.Now())
		if err != nil {
			return err
		}
	}

	tagsQuery := `INSERT INTO poll_tags (poll_id, tag, created_at) VALUES ($1, $2, $3)`
	for _, tag := range tags {
		_, err = tx.ExecContext(ctx, tagsQuery, poll.ID, tag, time.Now())
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func closeRows(rows *sql.Rows, logger *zap.Logger) {
	if err := rows.Close(); err != nil {
		logger.Error("Failed to close rows", zap.Error(err))
	}
}

func (r *PostgresRepository) GetPollByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	pollQuery := `SELECT id, title, created_at, updated_at FROM polls WHERE id = $1`
	poll := &domain.Poll{}
	err := r.db.QueryRowContext(ctx, pollQuery, id).Scan(&poll.ID, &poll.Title, &poll.CreatedAt, &poll.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	optionsQuery := `SELECT id, option_text, option_index, created_at FROM poll_options WHERE poll_id = $1 ORDER BY option_index`
	rows, err := r.db.QueryContext(ctx, optionsQuery, id)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	for rows.Next() {
		option := domain.Option{PollID: id}
		err := rows.Scan(&option.ID, &option.OptionText, &option.OptionIndex, &option.CreatedAt)
		if err != nil {
			return nil, err
		}
		poll.Options = append(poll.Options, option)
	}

	tagsQuery := `SELECT tag FROM poll_tags WHERE poll_id = $1`
	rows, err = r.db.QueryContext(ctx, tagsQuery, id)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	for rows.Next() {
		var tag string
		err := rows.Scan(&tag)
		if err != nil {
			return nil, err
		}
		poll.Tags = append(poll.Tags, tag)
	}

	return poll, nil
}

func (r *PostgresRepository) GetPollsForFeed(ctx context.Context, query domain.FeedQuery) ([]*domain.Poll, error) {
	offset := (query.Page - 1) * query.Limit

	baseQuery := `
		SELECT DISTINCT p.id, p.title, p.created_at, p.updated_at
		FROM polls p
		LEFT JOIN poll_tags pt ON p.id = pt.poll_id
		WHERE p.id NOT IN (
			SELECT poll_id FROM votes WHERE user_id = $1
			UNION
			SELECT poll_id FROM skips WHERE user_id = $1
		)
	`

	args := []interface{}{query.UserID}
	argCount := 2

	if query.Tag != "" {
		baseQuery += ` AND pt.tag = $` + string(rune('0'+argCount))
		args = append(args, query.Tag)
		argCount++
	}

	baseQuery += ` ORDER BY p.created_at DESC LIMIT $` + string(rune('0'+argCount)) + ` OFFSET $` + string(rune('0'+argCount+1))
	args = append(args, query.Limit, offset)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	var polls []*domain.Poll
	for rows.Next() {
		poll := &domain.Poll{}
		err := rows.Scan(&poll.ID, &poll.Title, &poll.CreatedAt, &poll.UpdatedAt)
		if err != nil {
			return nil, err
		}

		poll, err = r.GetPollByID(ctx, poll.ID)
		if err != nil {
			return nil, err
		}
		polls = append(polls, poll)
	}

	return polls, nil
}

func (r *PostgresRepository) GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	query := `
		SELECT po.option_text, COUNT(v.id) as vote_count
		FROM poll_options po
		LEFT JOIN votes v ON po.id = v.option_id
		WHERE po.poll_id = $1
		GROUP BY po.option_text, po.option_index
		ORDER BY po.option_index
	`

	rows, err := r.db.QueryContext(ctx, query, pollID)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, r.logger)

	stats := &domain.PollStats{
		PollID: pollID,
		Votes:  make([]domain.OptionStats, 0),
	}

	for rows.Next() {
		var optionStat domain.OptionStats
		err := rows.Scan(&optionStat.Option, &optionStat.Count)
		if err != nil {
			return nil, err
		}
		stats.Votes = append(stats.Votes, optionStat)
	}

	return stats, nil
}

func (r *PostgresRepository) CreateVote(ctx context.Context, vote *domain.Vote) error {
	query := `INSERT INTO votes (id, poll_id, user_id, option_id, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, vote.ID, vote.PollID, vote.UserID, vote.OptionID, vote.CreatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrAlreadyVoted
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) HasUserVoted(ctx context.Context, pollID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM votes WHERE poll_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, pollID, userID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) GetUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	query := `SELECT vote_count FROM user_daily_votes WHERE user_id = $1 AND vote_date = $2`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, date).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return count, err
}

func (r *PostgresRepository) IncrementUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) error {
	query := `
		INSERT INTO user_daily_votes (user_id, vote_date, vote_count)
		VALUES ($1, $2, 1)
		ON CONFLICT (user_id, vote_date)
		DO UPDATE SET vote_count = user_daily_votes.vote_count + 1
		WHERE user_daily_votes.vote_count < $3
	`
	result, err := r.db.ExecContext(ctx, query, userID, date, domain.MaxDailyVotes)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.ErrDailyVoteLimitExceeded
	}

	return nil
}

func (r *PostgresRepository) CreateSkip(ctx context.Context, skip *domain.Skip) error {
	query := `INSERT INTO skips (id, poll_id, user_id, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, skip.ID, skip.PollID, skip.UserID, skip.CreatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrAlreadySkipped
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) HasUserSkipped(ctx context.Context, pollID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM skips WHERE poll_id = $1 AND user_id = $2)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, pollID, userID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx, r.logger)

	if err := fn(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func rollbackTx(tx *sql.Tx, logger *zap.Logger) {
	if err := tx.Rollback(); err != nil {
		logger.Error("Failed to rollback transaction", zap.Error(err))
	}
}
