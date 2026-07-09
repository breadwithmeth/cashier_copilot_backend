package repository

import (
	"context"
	"fmt"

	"cashier_copilot_backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo handles persistence of users used by local authentication.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// List returns all users.
func (r *UserRepo) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, username, password_hash, role, pos_id, is_active, created_at
		 FROM users
		 ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.PosID, &u.IsActive, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// FindByUsername returns a user by username.
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, pos_id, is_active, created_at
		 FROM users
		 WHERE username = $1
		 LIMIT 1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.PosID, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by username: %w", err)
	}
	return &u, nil
}

// FindByID returns a user by ID.
func (r *UserRepo) FindByID(ctx context.Context, id int64) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, username, password_hash, role, pos_id, is_active, created_at
		 FROM users
		 WHERE id = $1
		 LIMIT 1`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.PosID, &u.IsActive, &u.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &u, nil
}

// Insert creates a user.
func (r *UserRepo) Insert(ctx context.Context, user *model.User) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, role, pos_id, is_active)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		user.Username, user.PasswordHash, user.Role, user.PosID, user.IsActive,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert user: %w", err)
	}
	return id, nil
}

// InsertIfMissing creates a user when username does not already exist.
func (r *UserRepo) InsertIfMissing(ctx context.Context, user *model.User) (bool, error) {
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO users (username, password_hash, role, pos_id, is_active)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (username) DO NOTHING`,
		user.Username, user.PasswordHash, user.Role, user.PosID, user.IsActive,
	)
	if err != nil {
		return false, fmt.Errorf("insert user if missing: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
