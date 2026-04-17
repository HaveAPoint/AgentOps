package model

import (
	"context"
	"database/sql"
	"time"
)

type TaskStatusHistory struct {
	ID         int64
	TaskID     int64
	FromStatus sql.NullString
	ToStatus   string
	Action     string
	ActorID    string
	ActorRole  string
	Reason     string
	CreatedAt  time.Time
}

type TaskStatusHistoryModel struct {
	db *sql.DB
}

func NewTaskStatusHistoryModel(db *sql.DB) *TaskStatusHistoryModel {
	return &TaskStatusHistoryModel{db: db}
}

func (m *TaskStatusHistoryModel) Insert(ctx context.Context, exec DBTX, history *TaskStatusHistory) (int64, error) {
	query := `
		INSERT INTO task_status_histories (
			task_id,
			from_status,
			to_status,
			action,
			actor_id,
			actor_role,
			reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	var id int64
	var createdAt time.Time

	err := exec.QueryRowContext(
		ctx,
		query,
		history.TaskID,
		history.FromStatus,
		history.ToStatus,
		history.Action,
		history.ActorID,
		history.ActorRole,
		history.Reason,
	).Scan(&id, &createdAt)
	if err != nil {
		return 0, err
	}

	history.ID = id
	history.CreatedAt = createdAt

	return id, nil
}

func (m *TaskStatusHistoryModel) ListByTaskID(ctx context.Context, taskID int64) ([]TaskStatusHistory, error) {
	query := `
		SELECT
			id,
			task_id,
			from_status,
			to_status,
			action,
			actor_id,
			actor_role,
			reason,
			created_at
		FROM task_status_histories
		WHERE task_id = $1
		ORDER BY created_at ASC, id ASC
	`

	rows, err := m.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TaskStatusHistory, 0)
	for rows.Next() {
		var history TaskStatusHistory
		if err := rows.Scan(
			&history.ID,
			&history.TaskID,
			&history.FromStatus,
			&history.ToStatus,
			&history.Action,
			&history.ActorID,
			&history.ActorRole,
			&history.Reason,
			&history.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, history)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
