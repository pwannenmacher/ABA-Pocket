package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"aba-pocket/internal/models"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, username, email, passwordHash string) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, password_hash, is_active, created_at, updated_at`,
		username, email, passwordHash,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM users WHERE username = $1`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, is_active, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepository) List(ctx context.Context) ([]*models.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, username, email, is_active, created_at, updated_at
		FROM users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// Session management

func (r *UserRepository) CreateSession(ctx context.Context, sessionID string, userID int64, duration time.Duration) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES ($1, $2, $3)`,
		sessionID, userID, time.Now().Add(duration),
	)
	return err
}

func (r *UserRepository) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	s := &models.Session{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, created_at, expires_at
		FROM sessions WHERE id = $1 AND expires_at > NOW()`, sessionID,
	).Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return s, nil
}

func (r *UserRepository) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}
