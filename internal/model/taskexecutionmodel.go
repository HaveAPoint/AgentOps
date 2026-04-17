package model

import (
	"context"
	"database/sql"
	"time"
)

type TaskExecution struct {
	ID            int64
	TaskID        int64
	OperatorId    string
	Status        string
	StartedAt     sql.NullTime
	FinishedAt    sql.NullTime
	ResultSummary string
	ErrorMessage  string
	CreatedAt     time.Time
}

type FinishExecutionParams struct {
	Status        string
	FinishedAt    time.Time
	ResultSummary string
	ErrorMessage  string
}

type TaskExecutionModel struct {
	db *sql.DB
}

func NewTaskExecutionModel(db *sql.DB) *TaskExecutionModel {
	return &TaskExecutionModel{db: db}
}

func (m *TaskExecutionModel) Insert(ctx context.Context, exec DBTX, taskExecution *TaskExecution) (int64, error) {
	query := `
		INSERT INTO task_executions (
			task_id,
			operator_id,
			status,
			started_at,
			finished_at,
			result_summary,
			error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	var finishedAt any
	if taskExecution.FinishedAt.Valid {
		finishedAt = taskExecution.FinishedAt.Time
	}

	var id int64
	var createdAt time.Time

	err := exec.QueryRowContext(
		ctx,
		query,
		taskExecution.TaskID,
		taskExecution.OperatorId,
		taskExecution.Status,
		taskExecution.StartedAt,
		finishedAt,
		taskExecution.ResultSummary,
		taskExecution.ErrorMessage,
	).Scan(&id, &createdAt)
	if err != nil {
		return 0, err
	}

	taskExecution.ID = id
	taskExecution.CreatedAt = createdAt

	return id, nil
}

func (m *TaskExecutionModel) FindLatestRunningByTaskIDForUpdate(ctx context.Context, exec DBTX, taskID int64) (*TaskExecution, error) {
	query := `
		SELECT
			id,
			task_id,
			operator_id,
			status,
			started_at,
			finished_at,
			result_summary,
			error_message,
			created_at
		FROM task_executions
		WHERE task_id = $1 AND status = 'running'
		ORDER BY id DESC
		LIMIT 1
		FOR UPDATE
	`

	var taskExecution TaskExecution
	err := exec.QueryRowContext(ctx, query, taskID).Scan(
		&taskExecution.ID,
		&taskExecution.TaskID,
		&taskExecution.OperatorId,
		&taskExecution.Status,
		&taskExecution.StartedAt,
		&taskExecution.FinishedAt,
		&taskExecution.ResultSummary,
		&taskExecution.ErrorMessage,
		&taskExecution.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &taskExecution, nil
}
func (m *TaskExecutionModel) Finish(ctx context.Context, exec DBTX, id int64, params FinishExecutionParams) error {
	query := `
		UPDATE task_executions
		SET
			status = $2,
			finished_at = $3,
			result_summary = $4,
			error_message = $5
		WHERE id = $1
	`

	_, err := exec.ExecContext(
		ctx,
		query,
		id,
		params.Status,
		params.FinishedAt,
		params.ResultSummary,
		params.ErrorMessage,
	)
	return err
}

func (m *TaskExecutionModel) ListByTaskID(ctx context.Context, taskID int64) ([]TaskExecution, error) {
	query := `
		SELECT
			id,
			task_id,
			operator_id,
			status,
			started_at,
			finished_at,
			result_summary,
			error_message,
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
			&exec.OperatorId,
			&exec.Status,
			&exec.StartedAt,
			&exec.FinishedAt,
			&exec.ResultSummary,
			&exec.ErrorMessage,
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
