package model

import (
	"context"
	"database/sql"
	"time"
)

type ApprovalRecord struct {
	ID         int64
	TaskID     int64
	ReviewerId string
	Decision   string
	Reason     string
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
			reviewer_id,
			decision,
			reason
		) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	var id int64
	var createdAt time.Time

	err := exec.QueryRowContext(
		ctx,
		query,
		record.TaskID,
		record.ReviewerId,
		record.Decision,
		record.Reason,
	).Scan(&id, &createdAt)
	if err != nil {
		return 0, err
	}

	record.ID = id
	record.CreatedAt = createdAt

	return id, nil
}
