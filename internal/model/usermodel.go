package model

import (
	"context"
	"database/sql"
	"time"
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	SystemRole   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserModel struct {
	db *sql.DB
}

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{db: db}
}

func (m *UserModel) Insert(ctx context.Context, exec DBTX, user *User) error {
	query := `
		INSERT INTO users (
			id,
			username,
			password_hash,
			system_role
		) VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`

	return exec.QueryRowContext(
		ctx,
		query,
		user.ID,
		user.Username,
		user.PasswordHash,
		user.SystemRole,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

func (m *UserModel) FindByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT
			id,
			username,
			password_hash,
			system_role,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.SystemRole,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (m *UserModel) FindByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT
			id,
			username,
			password_hash,
			system_role,
			created_at,
			updated_at
		FROM users
		WHERE username = $1
	`

	var user User
	err := m.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.SystemRole,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
