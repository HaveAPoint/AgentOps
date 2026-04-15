// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package tasks

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"agentops/internal/model"
	"agentops/internal/svc"
	"agentops/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApproveTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApproveTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApproveTaskLogic {
	return &ApproveTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApproveTaskLogic) ApproveTask(req *types.ApproveTaskReq) (resp *types.ApproveTaskResp, err error) {
	idText := strings.TrimSpace(req.Id)
	if idText == "" {
		return nil, ErrTaskIDRequired
	}

	reviewerID := strings.TrimSpace(req.ReviewerId)
	if reviewerID == "" {
		return nil, errors.New("reviewerId is required")
	}

	taskID, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || taskID <= 0 {
		return nil, ErrInvalidTaskID
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	task, err := l.svcCtx.TaskModel.FindByIDForUpdate(l.ctx, tx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if task.Status != TaskStatusWaitingApproval {
		return nil, ErrTaskNotWaitingApproval
	}

	if _, err = l.svcCtx.TaskModel.UpdateStatus(l.ctx, tx, taskID, TaskStatusPending); err != nil {
		return nil, err
	}

	if _, err = l.svcCtx.ApprovalRecordModel.Insert(l.ctx, tx, &model.ApprovalRecord{
		TaskID:     taskID,
		ApprovedBy: reviewerID,
		Comment:    strings.TrimSpace(req.Comment),
	}); err != nil {
		return nil, err
	}

	maxStep, err := l.svcCtx.AuditLogModel.GetMaxStep(l.ctx, tx, taskID)
	if err != nil {
		return nil, err
	}

	if _, err = l.svcCtx.AuditLogModel.Insert(l.ctx, tx, &model.AuditLog{
		TaskID:     taskID,
		Step:       maxStep + 1,
		Level:      "info",
		Message:    "task approved",
		ToolName:   "api",
		OccurredAt: time.Now().UTC(),
	}); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &types.ApproveTaskResp{
		Id:     strconv.FormatInt(taskID, 10),
		Status: TaskStatusPending,
	}, nil
}
