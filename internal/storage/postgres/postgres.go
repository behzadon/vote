package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/behzadon/vote/internal/domain"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

type Repository struct {
	db     *sql.DB
	redis  *redis.Client
	logger *zap.Logger
}

func NewRepository(db *sql.DB, redis *redis.Client, logger *zap.Logger) *Repository {
	return &Repository{
		db:     db,
		redis:  redis,
		logger: logger,
	}
}

func (r *Repository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, username, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Username, user.Email, user.Password,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrEmailAlreadyExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT id, username, email, password, created_at, updated_at FROM users WHERE email = $1`
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users 
		SET username = $1, email = $2, password = $3, updated_at = $4
		WHERE id = $5
	`
	_, err := r.db.ExecContext(ctx, query,
		user.Username, user.Email, user.Password,
		user.UpdatedAt, user.ID,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrEmailAlreadyExists
		}
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *Repository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (r *Repository) CreatePoll(ctx context.Context, poll *domain.Poll, options []string, tags []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
				r.logger.Error("Failed to rollback transaction", zap.Error(err))
			}
		}
	}()

	query := `
		INSERT INTO polls (id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
	err = tx.QueryRowContext(ctx, query,
		poll.ID, poll.Title, time.Now().UTC(), time.Now().UTC(),
	).Scan(&poll.ID)
	if err != nil {
		return fmt.Errorf("insert poll: %w", err)
	}

	optionsQuery := `
		INSERT INTO poll_options (id, poll_id, option_text, option_index, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	for i, optionText := range options {
		optionID := uuid.New()
		var id uuid.UUID
		err = tx.QueryRowContext(ctx, optionsQuery,
			optionID, poll.ID, optionText, i, time.Now().UTC(),
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("insert option %d: %w", i, err)
		}
		poll.Options = append(poll.Options, domain.Option{
			ID:          id,
			PollID:      poll.ID,
			OptionText:  optionText,
			OptionIndex: i,
			CreatedAt:   time.Now().UTC(),
		})
	}

	if len(tags) > 0 {
		tagsQuery := `
			INSERT INTO poll_tags (poll_id, tag)
			VALUES ($1, $2)`
		for _, tag := range tags {
			_, err = tx.ExecContext(ctx, tagsQuery, poll.ID, tag)
			if err != nil {
				return fmt.Errorf("insert tag %s: %w", tag, err)
			}
		}
		poll.Tags = tags
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true
	return nil
}

func (r *Repository) GetCachedPoll(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	key := "poll:" + id.String()
	data, err := r.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get cached poll: %w", err)
	}
	var poll domain.Poll
	if err := json.Unmarshal(data, &poll); err != nil {
		return nil, fmt.Errorf("unmarshal cached poll: %w", err)
	}
	return &poll, nil
}

func (r *Repository) SetCachedPoll(ctx context.Context, poll *domain.Poll) error {
	key := "poll:" + poll.ID.String()
	data, err := json.Marshal(poll)
	if err != nil {
		return fmt.Errorf("marshal poll: %w", err)
	}
	if err := r.redis.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("cache poll: %w", err)
	}
	return nil
}

func (r *Repository) GetPollByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	poll, err := r.GetCachedPoll(ctx, id)
	if err == nil && poll != nil {
		return poll, nil
	}
	query := `
		SELECT p.id, p.title, p.created_at, p.updated_at
		FROM polls p
		WHERE p.id = $1`
	poll = &domain.Poll{ID: id}
	err = r.db.QueryRowContext(ctx, query, id).Scan(
		&poll.ID, &poll.Title, &poll.CreatedAt, &poll.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get poll: %w", err)
	}

	optionsQuery := `
		SELECT id, option_text, created_at
		FROM poll_options
		WHERE poll_id = $1
		ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, optionsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("get options: %w", err)
	}
	defer closeRows(rows, r.logger)

	for rows.Next() {
		var option domain.Option
		err = rows.Scan(&option.ID, &option.OptionText, &option.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan option: %w", err)
		}
		option.PollID = id
		poll.Options = append(poll.Options, option)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate options: %w", err)
	}

	tagsQuery := `
		SELECT tag
		FROM poll_tags
		WHERE poll_id = $1
		ORDER BY tag`
	rows, err = r.db.QueryContext(ctx, tagsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}
	defer closeRows(rows, r.logger)

	for rows.Next() {
		var tag string
		err = rows.Scan(&tag)
		if err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		poll.Tags = append(poll.Tags, tag)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	_ = r.SetCachedPoll(ctx, poll)

	return poll, nil
}

func (r *Repository) GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) ([]domain.Poll, int, error) {
	baseQuery := `
		FROM polls p
		WHERE NOT EXISTS (
			SELECT 1 FROM votes v WHERE v.poll_id = p.id AND v.user_id = $1
		)
		AND NOT EXISTS (
			SELECT 1 FROM skips s WHERE s.poll_id = p.id AND s.user_id = $1
		)`
	args := []interface{}{userID}
	argCount := 1

	if tag != "" {
		argCount++
		baseQuery += fmt.Sprintf(`
			AND EXISTS (
				SELECT 1 FROM poll_tags pt WHERE pt.poll_id = p.id AND pt.tag = $%d
			)`, argCount)
		args = append(args, tag)
	}

	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("get total count: %w", err)
	}

	query := `
		SELECT p.id, p.title, p.created_at, p.updated_at
		` + baseQuery + `
		ORDER BY p.created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argCount+1) + `
		OFFSET $` + fmt.Sprintf("%d", argCount+2)
	args = append(args, limit, (page-1)*limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("get polls: %w", err)
	}
	defer closeRows(rows, r.logger)

	var polls []domain.Poll
	for rows.Next() {
		var poll domain.Poll
		err = rows.Scan(&poll.ID, &poll.Title, &poll.CreatedAt, &poll.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan poll: %w", err)
		}

		optionsQuery := `
			SELECT id, option_text, created_at
			FROM poll_options
			WHERE poll_id = $1
			ORDER BY created_at`
		optionRows, err := r.db.QueryContext(ctx, optionsQuery, poll.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("get options: %w", err)
		}
		defer closeRows(optionRows, r.logger)

		for optionRows.Next() {
			var option domain.Option
			err = optionRows.Scan(&option.ID, &option.OptionText, &option.CreatedAt)
			if err != nil {
				return nil, 0, fmt.Errorf("scan option: %w", err)
			}
			option.PollID = poll.ID
			poll.Options = append(poll.Options, option)
		}
		if err = optionRows.Err(); err != nil {
			return nil, 0, fmt.Errorf("iterate options: %w", err)
		}

		tagsQuery := `
			SELECT tag
			FROM poll_tags
			WHERE poll_id = $1
			ORDER BY tag`
		tagRows, err := r.db.QueryContext(ctx, tagsQuery, poll.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("get tags: %w", err)
		}
		defer closeRows(tagRows, r.logger)

		for tagRows.Next() {
			var tag string
			err = tagRows.Scan(&tag)
			if err != nil {
				return nil, 0, fmt.Errorf("scan tag: %w", err)
			}
			poll.Tags = append(poll.Tags, tag)
		}
		if err = tagRows.Err(); err != nil {
			return nil, 0, fmt.Errorf("iterate tags: %w", err)
		}

		polls = append(polls, poll)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate polls: %w", err)
	}

	return polls, total, nil
}

func (r *Repository) GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	query := `
		SELECT po.option_text, COUNT(v.id) as vote_count
		FROM poll_options po
		LEFT JOIN votes v ON v.option_id = po.id
		WHERE po.poll_id = $1
		GROUP BY po.option_text, po.created_at
		ORDER BY po.created_at`
	rows, err := r.db.QueryContext(ctx, query, pollID)
	if err != nil {
		return nil, fmt.Errorf("get poll stats: %w", err)
	}
	defer closeRows(rows, r.logger)

	stats := &domain.PollStats{
		PollID: pollID,
		Votes:  make([]domain.OptionStats, 0),
	}
	for rows.Next() {
		var optionStats domain.OptionStats
		err = rows.Scan(&optionStats.Option, &optionStats.Count)
		if err != nil {
			return nil, fmt.Errorf("scan option stats: %w", err)
		}
		stats.Votes = append(stats.Votes, optionStats)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate option stats: %w", err)
	}

	return stats, nil
}

func (r *Repository) CreateVote(ctx context.Context, pollID, userID, optionID uuid.UUID) error {
	query := `
		INSERT INTO votes (id, poll_id, user_id, option_id, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query,
		uuid.New(), pollID, userID, optionID, time.Now().UTC(),
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrAlreadyVoted
		}
		return fmt.Errorf("create vote: %w", err)
	}

	poll, err := r.GetPollByID(ctx, pollID)
	if err == nil {
		_ = r.SetCachedPoll(ctx, poll)
	} else {
		r.logger.Warn("Failed to re-cache poll after vote", zap.Error(err))
	}

	return nil
}

func (r *Repository) HasVoted(ctx context.Context, pollID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM votes
			WHERE poll_id = $1 AND user_id = $2
		)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, pollID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check vote: %w", err)
	}
	return exists, nil
}

func (r *Repository) GetUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	query := `
		SELECT vote_count
		FROM user_daily_votes
		WHERE user_id = $1 AND vote_date = $2`
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, date).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get daily vote count: %w", err)
	}
	return count, nil
}

func (r *Repository) IncrementUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) error {
	query := `
		INSERT INTO user_daily_votes (user_id, vote_date, vote_count, created_at, updated_at)
		VALUES ($1, $2, 1, $3, $3)
		ON CONFLICT (user_id, vote_date) DO UPDATE
		SET vote_count = user_daily_votes.vote_count + 1,
			updated_at = $3`
	_, err := r.db.ExecContext(ctx, query, userID, date, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("increment daily vote count: %w", err)
	}
	return nil
}

func (r *Repository) CreateSkip(ctx context.Context, pollID, userID uuid.UUID) error {
	query := `
		INSERT INTO skips (id, poll_id, user_id, created_at)
		VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query,
		uuid.New(), pollID, userID, time.Now().UTC(),
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrAlreadySkipped
		}
		return fmt.Errorf("create skip: %w", err)
	}
	return nil
}

func (r *Repository) HasSkipped(ctx context.Context, pollID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM skips
			WHERE poll_id = $1 AND user_id = $2
		)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, pollID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check skip: %w", err)
	}
	return exists, nil
}

func (r *Repository) GetCachedPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	key := fmt.Sprintf("poll:stats:%s", pollID)
	data, err := r.redis.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get cached stats: %w", err)
	}

	var stats domain.PollStats
	err = json.Unmarshal(data, &stats)
	if err != nil {
		return nil, fmt.Errorf("unmarshal cached stats: %w", err)
	}
	return &stats, nil
}

func (r *Repository) SetCachedPollStats(ctx context.Context, pollID uuid.UUID, stats *domain.PollStats) error {
	key := fmt.Sprintf("poll:stats:%s", pollID)
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal stats: %w", err)
	}

	err = r.redis.Set(ctx, key, data, 5*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("cache stats: %w", err)
	}
	return nil
}

func (r *Repository) InvalidatePollStatsCache(ctx context.Context, pollID uuid.UUID) error {
	key := fmt.Sprintf("poll:stats:%s", pollID)
	err := r.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("invalidate cache: %w", err)
	}
	return nil
}

func (r *Repository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer rollbackTx(tx, r.logger)

	txCtx := context.WithValue(ctx, txKey{}, tx)

	err = fn(txCtx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func rollbackTx(tx *sql.Tx, logger *zap.Logger) {
	if err := tx.Rollback(); err != nil {
		logger.Error("Failed to rollback transaction", zap.Error(err))
	}
}

func closeRows(rows *sql.Rows, logger *zap.Logger) {
	if err := rows.Close(); err != nil {
		logger.Error("Failed to close rows", zap.Error(err))
	}
}

type txKey struct{}

func (r *Repository) GetUserVotes(ctx context.Context, userID uuid.UUID, page, limit int) ([]domain.Vote, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM votes
		WHERE user_id = $1`
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("get vote count: %w", err)
	}

	query := `
		SELECT v.id, v.poll_id, v.user_id, v.option_id, v.created_at,
			   p.title as poll_title,
			   po.option_text as option_text
		FROM votes v
		JOIN polls p ON v.poll_id = p.id
		JOIN poll_options po ON v.option_id = po.id
		WHERE v.user_id = $1
		ORDER BY v.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, fmt.Errorf("get votes: %w", err)
	}
	defer closeRows(rows, r.logger)

	var votes []domain.Vote
	for rows.Next() {
		var vote domain.Vote
		var pollTitle, optionText string
		err = rows.Scan(
			&vote.ID, &vote.PollID, &vote.UserID, &vote.OptionID, &vote.CreatedAt,
			&pollTitle, &optionText,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan vote: %w", err)
		}
		votes = append(votes, vote)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate votes: %w", err)
	}

	return votes, total, nil
}

func (r *Repository) GetVoteByID(ctx context.Context, voteID uuid.UUID) (*domain.Vote, error) {
	query := `
		SELECT v.id, v.poll_id, v.user_id, v.option_id, v.created_at
		FROM votes v
		WHERE v.id = $1`

	var vote domain.Vote
	err := r.db.QueryRowContext(ctx, query, voteID).Scan(
		&vote.ID, &vote.PollID, &vote.UserID, &vote.OptionID, &vote.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get vote by id: %w", err)
	}
	return &vote, nil
}

func (r *Repository) UpdateVote(ctx context.Context, voteID, userID, optionID uuid.UUID) error {
	vote, err := r.GetVoteByID(ctx, voteID)
	if err != nil {
		return err
	}
	if vote.UserID != userID {
		return domain.ErrUnauthorized
	}

	query := `
		SELECT EXISTS (
			SELECT 1 FROM poll_options po
			WHERE po.id = $1 AND po.poll_id = (
				SELECT poll_id FROM votes WHERE id = $2
			)
		)`
	var exists bool
	err = r.db.QueryRowContext(ctx, query, optionID, voteID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("verify option: %w", err)
	}
	if !exists {
		return domain.ErrInvalidOption
	}

	updateQuery := `
		UPDATE votes
		SET option_id = $1
		WHERE id = $2 AND user_id = $3`

	result, err := r.db.ExecContext(ctx, updateQuery, optionID, voteID, userID)
	if err != nil {
		return fmt.Errorf("update vote: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}

	if err := r.InvalidatePollStatsCache(ctx, vote.PollID); err != nil {
		r.logger.Warn("Failed to invalidate poll stats cache after vote update",
			zap.Error(err),
			zap.String("poll_id", vote.PollID.String()),
		)
	}

	return nil
}

func (r *Repository) DeleteVote(ctx context.Context, voteID, userID uuid.UUID) error {
	query := `DELETE FROM votes WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, voteID, userID)
	if err != nil {
		return fmt.Errorf("delete vote: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete vote rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrUnauthorized
	}
	return nil
}
