package model

import (
	"context"
	"database/sql"
	"time"
)

type ApprovalRecord struct {
	ID         int64
	TaskID     int64
	ApprovedBy string
	Comment    string
	CreatedAt  time.Time
}

type ApprovalRecordModel struct {
	db *sql.DB
}

func NewApprovalRecordModel(db *sql.DB) *ApprovalRecordModel {
	return &ApprovalRecordModel{db: db}
}

func (m *ApprovalRecordModel) Insert(ctx context.Context, exec DBTX, record *ApprovalRecord) (int64, error) {
	query := `
		INSERT INTO approval_records (
			task_id,
			approved_by,
			comment
		) VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	var id int64
	var createdAt time.Time

	err := exec.QueryRowContext(
		ctx,
		query,
		record.TaskID,
		record.ApprovedBy,
		record.Comment,
	).Scan(&id, &createdAt)
	if err != nil {
		return 0, err
	}

	record.ID = id
	record.CreatedAt = createdAt

	return id, nil
}
