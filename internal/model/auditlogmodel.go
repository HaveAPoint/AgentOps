package model

import (
	"context"
	"database/sql"
	"time"
)

type AuditLog struct {
	ID         int64
	TaskID     int64
	Step       int64
	Level      string
	Message    string
	ToolName   string
	OccurredAt time.Time
	CreatedAt  time.Time
}

type AuditLogModel struct {
	db *sql.DB
}

func NewAuditLogModel(db *sql.DB) *AuditLogModel {
	return &AuditLogModel{db: db}
}

func (m *AuditLogModel) Insert(ctx context.Context, exec DBTX, log *AuditLog) (int64, error) {
	query := `
		INSERT INTO audit_logs (
			task_id,
			step,
			level,
			message,
			tool_name,
			occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	var id int64
	var createdAt time.Time

	err := exec.QueryRowContext(
		ctx,
		query,
		log.TaskID,
		log.Step,
		log.Level,
		log.Message,
		log.ToolName,
		log.OccurredAt,
	).Scan(&id, &createdAt)
	if err != nil {
		return 0, err
	}

	log.ID = id
	log.CreatedAt = createdAt

	return id, nil
}

func (m *AuditLogModel) GetMaxStep(ctx context.Context, exec DBTX, taskID int64) (int64, error) {
	query := `
		SELECT COALESCE(MAX(step), 0)
		FROM audit_logs
		WHERE task_id = $1
	`

	var step int64
	err := exec.QueryRowContext(ctx, query, taskID).Scan(&step)
	if err != nil {
		return 0, err
	}

	return step, nil
}

func (m *AuditLogModel) ListByTaskID(ctx context.Context, taskID int64) ([]AuditLog, error) {
	query := `
		SELECT
			id,
			task_id,
			step,
			level,
			message,
			tool_name,
			occurred_at,
			created_at
		FROM audit_logs
		WHERE task_id = $1
		ORDER BY step ASC, id ASC
	`

	rows, err := m.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AuditLog, 0)
	for rows.Next() {
		var log AuditLog
		if err := rows.Scan(
			&log.ID,
			&log.TaskID,
			&log.Step,
			&log.Level,
			&log.Message,
			&log.ToolName,
			&log.OccurredAt,
			&log.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
