package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/behzadon/vote/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(dsn string) (domain.Repository, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
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
	return err
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &user, err
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.GetContext(ctx, &user, query, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &user, err
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
	return err
}

func (r *Repository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) CreatePoll(ctx context.Context, poll *domain.Poll, options []string, tags []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	pollQuery := `
		INSERT INTO polls (id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.ExecContext(ctx, pollQuery,
		poll.ID, poll.Title, poll.CreatedAt, poll.UpdatedAt,
	)
	if err != nil {
		return err
	}

	optionQuery := `
		INSERT INTO poll_options (id, poll_id, option_text, option_index, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, opt := range poll.Options {
		_, err = tx.ExecContext(ctx, optionQuery,
			opt.ID, poll.ID, opt.OptionText, opt.OptionIndex, opt.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	tagQuery := `INSERT INTO poll_tags (poll_id, tag) VALUES ($1, $2)`
	for _, tag := range tags {
		_, err = tx.ExecContext(ctx, tagQuery, poll.ID, tag)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) GetPollByID(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	var poll domain.Poll
	pollQuery := `SELECT * FROM polls WHERE id = $1`
	err := r.db.GetContext(ctx, &poll, pollQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	optionsQuery := `SELECT * FROM poll_options WHERE poll_id = $1 ORDER BY option_index`
	err = r.db.SelectContext(ctx, &poll.Options, optionsQuery, id)
	if err != nil {
		return nil, err
	}

	tagsQuery := `SELECT tag FROM poll_tags WHERE poll_id = $1`
	err = r.db.SelectContext(ctx, &poll.Tags, tagsQuery, id)
	if err != nil {
		return nil, err
	}

	return &poll, nil
}

func (r *Repository) GetPollsForFeed(ctx context.Context, userID uuid.UUID, tag string, page, limit int) ([]domain.Poll, int, error) {
	var polls []domain.Poll
	var total int

	baseQuery := `
		SELECT DISTINCT p.* 
		FROM polls p
		LEFT JOIN poll_tags pt ON p.id = pt.poll_id
		LEFT JOIN votes v ON p.id = v.poll_id AND v.user_id = $1
		LEFT JOIN skips s ON p.id = s.poll_id AND s.user_id = $1
		WHERE v.id IS NULL AND s.id IS NULL
	`
	countQuery := `
		SELECT COUNT(DISTINCT p.id)
		FROM polls p
		LEFT JOIN poll_tags pt ON p.id = pt.poll_id
		LEFT JOIN votes v ON p.id = v.poll_id AND v.user_id = $1
		LEFT JOIN skips s ON p.id = s.poll_id AND s.user_id = $1
		WHERE v.id IS NULL AND s.id IS NULL
	`

	if tag != "" {
		baseQuery += ` AND pt.tag = $2`
		countQuery += ` AND pt.tag = $2`
	}

	baseQuery += ` ORDER BY p.created_at DESC LIMIT $3 OFFSET $4`

	var args []interface{}
	args = append(args, userID)
	if tag != "" {
		args = append(args, tag)
	}
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	err = r.db.SelectContext(ctx, &polls, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	for i := range polls {
		optionsQuery := `SELECT * FROM poll_options WHERE poll_id = $1 ORDER BY option_index`
		err = r.db.SelectContext(ctx, &polls[i].Options, optionsQuery, polls[i].ID)
		if err != nil {
			return nil, 0, err
		}

		tagsQuery := `SELECT tag FROM poll_tags WHERE poll_id = $1`
		err = r.db.SelectContext(ctx, &polls[i].Tags, tagsQuery, polls[i].ID)
		if err != nil {
			return nil, 0, err
		}
	}

	return polls, total, nil
}

func (r *Repository) GetPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	query := `
		SELECT po.option_text as option, COUNT(v.id) as count
		FROM poll_options po
		LEFT JOIN votes v ON po.id = v.option_id
		WHERE po.poll_id = $1
		GROUP BY po.option_text
		ORDER BY po.option_index
	`
	var stats domain.PollStats
	stats.PollID = pollID
	err := r.db.SelectContext(ctx, &stats.Votes, query, pollID)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *Repository) CreateVote(ctx context.Context, pollID, userID, optionID uuid.UUID) error {
	return r.WithTransaction(ctx, func(ctx context.Context) error {
		voteQuery := `
			INSERT INTO votes (id, poll_id, user_id, option_id, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`
		voteID := uuid.New()
		_, err := r.db.ExecContext(ctx, voteQuery,
			voteID, pollID, userID, optionID, time.Now().UTC(),
		)
		if err != nil {
			return err
		}

		dailyVoteQuery := `
			INSERT INTO user_daily_votes (id, user_id, vote_date, vote_count, created_at, updated_at)
			VALUES ($1, $2, $3, 1, $4, $4)
			ON CONFLICT (user_id, vote_date) 
			DO UPDATE SET vote_count = user_daily_votes.vote_count + 1, updated_at = $4
		`
		_, err = r.db.ExecContext(ctx, dailyVoteQuery,
			uuid.New(), userID, time.Now().UTC().Truncate(24*time.Hour), time.Now().UTC(),
		)
		return err
	})
}

func (r *Repository) HasVoted(ctx context.Context, pollID, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM votes WHERE poll_id = $1 AND user_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, pollID, userID)
	return exists, err
}

func (r *Repository) GetUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) (int, error) {
	var count int
	query := `
		SELECT vote_count 
		FROM user_daily_votes 
		WHERE user_id = $1 AND vote_date = $2
	`
	err := r.db.GetContext(ctx, &count, query, userID, date)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return count, err
}

func (r *Repository) IncrementUserDailyVoteCount(ctx context.Context, userID uuid.UUID, date time.Time) error {
	query := `
		INSERT INTO user_daily_votes (id, user_id, vote_date, vote_count, created_at, updated_at)
		VALUES ($1, $2, $3, 1, $4, $4)
		ON CONFLICT (user_id, vote_date) 
		DO UPDATE SET vote_count = user_daily_votes.vote_count + 1, updated_at = $4
	`
	_, err := r.db.ExecContext(ctx, query,
		uuid.New(), userID, date, time.Now().UTC(),
	)
	return err
}

func (r *Repository) CreateSkip(ctx context.Context, pollID, userID uuid.UUID) error {
	query := `
		INSERT INTO skips (id, poll_id, user_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query,
		uuid.New(), pollID, userID, time.Now().UTC(),
	)
	return err
}

func (r *Repository) HasSkipped(ctx context.Context, pollID, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM skips WHERE poll_id = $1 AND user_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, pollID, userID)
	return exists, err
}

func (r *Repository) GetCachedPollStats(ctx context.Context, pollID uuid.UUID) (*domain.PollStats, error) {
	return nil, domain.ErrNotFound
}

func (r *Repository) SetCachedPollStats(ctx context.Context, pollID uuid.UUID, stats *domain.PollStats) error {
	return nil
}

func (r *Repository) InvalidatePollStatsCache(ctx context.Context, pollID uuid.UUID) error {
	return nil
}

func (r *Repository) GetCachedPoll(ctx context.Context, id uuid.UUID) (*domain.Poll, error) {
	return nil, domain.ErrNotFound
}

func (r *Repository) SetCachedPoll(ctx context.Context, poll *domain.Poll) error {
	return nil
}

func (r *Repository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) DeleteVote(ctx context.Context, voteID, userID uuid.UUID) error {
	return nil
}

func (r *Repository) GetUserVotes(ctx context.Context, userID uuid.UUID, page, limit int) ([]domain.Vote, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM votes WHERE user_id = $1`
	err := r.db.GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT v.*, p.title as poll_title, po.option_text as option_text
		FROM votes v
		JOIN polls p ON v.poll_id = p.id
		JOIN poll_options po ON v.option_id = po.id
		WHERE v.user_id = $1
		ORDER BY v.created_at DESC
		LIMIT $2 OFFSET $3
	`
	offset := (page - 1) * limit
	rows, err := r.db.QueryxContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var votes []domain.Vote
	for rows.Next() {
		var vote domain.Vote
		var pollTitle, optionText string
		err := rows.Scan(
			&vote.ID,
			&vote.PollID,
			&vote.UserID,
			&vote.OptionID,
			&vote.CreatedAt,
			&pollTitle,
			&optionText,
		)
		if err != nil {
			return nil, 0, err
		}
		vote.PollTitle = pollTitle
		vote.OptionText = optionText
		votes = append(votes, vote)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return votes, total, nil
}

func (r *Repository) GetVoteByID(ctx context.Context, voteID uuid.UUID) (*domain.Vote, error) {
	return nil, nil
}

func (r *Repository) UpdateVote(ctx context.Context, voteID, userID, optionID uuid.UUID) error {
	return nil
}
