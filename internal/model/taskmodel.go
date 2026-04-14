package model

import (
	"context"
	"database/sql"
	"time"
)

type Task struct {
	ID               int64
	Title            string
	RepoPath         string
	Prompt           string
	Mode             string
	Status           string
	ApprovalRequired bool
	MaxSteps         int64
	CreatedBy        string
	GitBranch        string
	GitHeadCommit    string
	GitDirty         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type TaskModel struct {
	db *sql.DB
}

func NewTaskModel(db *sql.DB) *TaskModel {
	return &TaskModel{db: db}
}

func (m *TaskModel) Insert(ctx context.Context, exec DBTX, task *Task) (int64, error) {
	query := `
		INSERT INTO tasks (
			title,
			repo_path,
			prompt,
			mode,
			status,
			approval_required,
			max_steps,
			created_by,
			git_branch,
			git_head_commit,
			git_dirty
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	var id int64
	var createdAt time.Time
	var updatedAt time.Time

	err := exec.QueryRowContext(
		ctx,
		query,
		task.Title,
		task.RepoPath,
		task.Prompt,
		task.Mode,
		task.Status,
		task.ApprovalRequired,
		task.MaxSteps,
		task.CreatedBy,
		task.GitBranch,
		task.GitHeadCommit,
		task.GitDirty,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return 0, err
	}

	task.ID = id
	task.CreatedAt = createdAt
	task.UpdatedAt = updatedAt

	return id, nil
}

func (m *TaskModel) List(ctx context.Context) ([]Task, error) {
	query := `
		SELECT
			id,
			title,
			repo_path,
			mode,
			status,
			approval_required,
			created_by,
			git_branch,
			git_head_commit,
			git_dirty,
			created_at,
			updated_at
		FROM tasks
		ORDER BY id DESC
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.RepoPath,
			&task.Mode,
			&task.Status,
			&task.ApprovalRequired,
			&task.CreatedBy,
			&task.GitBranch,
			&task.GitHeadCommit,
			&task.GitDirty,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (m *TaskModel) FindByID(ctx context.Context, id int64) (*Task, error) {
	query := `
		SELECT
			id,
			title,
			repo_path,
			prompt,
			mode,
			status,
			approval_required,
			max_steps,
			created_by,
			git_branch,
			git_head_commit,
			git_dirty,
			created_at,
			updated_at
		FROM tasks
		WHERE id = $1
	`

	var task Task
	err := m.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		&task.RepoPath,
		&task.Prompt,
		&task.Mode,
		&task.Status,
		&task.ApprovalRequired,
		&task.MaxSteps,
		&task.CreatedBy,
		&task.GitBranch,
		&task.GitHeadCommit,
		&task.GitDirty,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (m *TaskModel) FindByIDForUpdate(ctx context.Context, exec DBTX, id int64) (*Task, error) {
	query := `
		SELECT
			id,
			title,
			repo_path,
			prompt,
			mode,
			status,
			approval_required,
			max_steps,
			created_by,
			git_branch,
			git_head_commit,
			git_dirty,
			created_at,
			updated_at
		FROM tasks
		WHERE id = $1
		FOR UPDATE
	`

	var task Task
	err := exec.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.Title,
		&task.RepoPath,
		&task.Prompt,
		&task.Mode,
		&task.Status,
		&task.ApprovalRequired,
		&task.MaxSteps,
		&task.CreatedBy,
		&task.GitBranch,
		&task.GitHeadCommit,
		&task.GitDirty,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (m *TaskModel) UpdateStatus(ctx context.Context, exec DBTX, id int64, status string) (time.Time, error) {
	query := `
		UPDATE tasks
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	var updatedAt time.Time
	err := exec.QueryRowContext(ctx, query, id, status).Scan(&updatedAt)
	if err != nil {
		return time.Time{}, err
	}

	return updatedAt, nil
}
