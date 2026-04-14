package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type TaskPolicy struct {
	ID           int64
	TaskID       int64
	AllowedPaths []string
	DeniedPaths  []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TaskPolicyModel struct {
	db *sql.DB
}

func NewTaskPolicyModel(db *sql.DB) *TaskPolicyModel {
	return &TaskPolicyModel{db: db}
}

func (m *TaskPolicyModel) Insert(ctx context.Context, exec DBTX, policy *TaskPolicy) (int64, error) {
	allowedPaths := policy.AllowedPaths
	if allowedPaths == nil {
		allowedPaths = []string{}
	}

	deniedPaths := policy.DeniedPaths
	if deniedPaths == nil {
		deniedPaths = []string{}
	}

	query := `
		INSERT INTO task_policies (
			task_id,
			allowed_paths,
			denied_paths
		) VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	var id int64
	var createdAt time.Time
	var updatedAt time.Time

	err := exec.QueryRowContext(
		ctx,
		query,
		policy.TaskID,
		pgtype.FlatArray[string](allowedPaths),
		pgtype.FlatArray[string](deniedPaths),
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return 0, err
	}

	policy.ID = id
	policy.CreatedAt = createdAt
	policy.UpdatedAt = updatedAt

	return id, nil
}

func (m *TaskPolicyModel) FindByTaskID(ctx context.Context, taskID int64) (*TaskPolicy, error) {
	query := `
		SELECT
			id,
			task_id,
			allowed_paths,
			denied_paths,
			created_at,
			updated_at
		FROM task_policies
		WHERE task_id = $1
	`

	var policy TaskPolicy
	typeMap := pgtype.NewMap()

	err := m.db.QueryRowContext(ctx, query, taskID).Scan(
		&policy.ID,
		&policy.TaskID,
		typeMap.SQLScanner(&policy.AllowedPaths),
		typeMap.SQLScanner(&policy.DeniedPaths),
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &policy, nil
}
