package model

import (
	"context"
	"database/sql"
	"time"
)

type TaskExecution struct {
	ID            int64
	TaskID        int64
	Status        string
	StartedAt     sql.NullTime
	FinishedAt    sql.NullTime
	ResultSummary string
	CreatedAt     time.Time
}

type TaskExecutionModel struct {
	db *sql.DB
}

func NewTaskExecutionModel(db *sql.DB) *TaskExecutionModel {
	return &TaskExecutionModel{db: db}
}

func (m *TaskExecutionModel) ListByTaskID(ctx context.Context, taskID int64) ([]TaskExecution, error) {
	query := `
		SELECT
			id,
			task_id,
			status,
			started_at,
			finished_at,
			result_summary,
			created_at
		FROM task_executions
		WHERE task_id = $1
		ORDER BY id DESC
	`

	rows, err := m.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TaskExecution, 0)
	for rows.Next() {
		var exec TaskExecution
		if err := rows.Scan(
			&exec.ID,
			&exec.TaskID,
			&exec.Status,
			&exec.StartedAt,
			&exec.FinishedAt,
			&exec.ResultSummary,
			&exec.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, exec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
