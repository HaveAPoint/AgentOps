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

type FailTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFailTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FailTaskLogic {
	return &FailTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FailTaskLogic) FailTask(req *types.FailTaskReq) (resp *types.FailTaskResp, err error) {
	idText := strings.TrimSpace(req.Id)
	if idText == "" {
		return nil, ErrTaskIDRequired
	}

	operatorID := strings.TrimSpace(req.OperatorId)
	if operatorID == "" {
		return nil, ErrOperatorIDRequired
	}

	errorMessage := strings.TrimSpace(req.ErrorMessage)
	if errorMessage == "" {
		return nil, ErrErrorMessageRequired
	}

	resultSummary := strings.TrimSpace(req.ResultSummary)

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

	if task.Status != TaskStatusRunning {
		return nil, ErrTaskNotRunning
	}

	execution, err := l.svcCtx.TaskExecutionModel.FindLatestRunningByTaskIDForUpdate(l.ctx, tx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRunningExecutionNotFound
		}
		return nil, err
	}

	if task.OperatorId.Valid && task.OperatorId.String != operatorID {
		return nil, ErrOperatorIDMismatch
	}

	if execution.OperatorId != operatorID {
		return nil, ErrExecutionOperatorMismatch
	}

	finishedAt := time.Now().UTC()

	if _, err = l.svcCtx.TaskModel.Fail(l.ctx, tx, taskID); err != nil {
		return nil, err
	}

	if _, err = l.svcCtx.TaskStatusHistoryModel.Insert(l.ctx, tx, &model.TaskStatusHistory{
		TaskID: taskID,
		FromStatus: sql.NullString{
			String: task.Status,
			Valid:  task.Status != "",
		},
		ToStatus:  TaskStatusFailed,
		Action:    "fail",
		ActorID:   operatorID,
		ActorRole: "operator",
		Reason:    errorMessage,
	}); err != nil {
		return nil, err
	}

	if err = l.svcCtx.TaskExecutionModel.Finish(
		l.ctx,
		tx,
		execution.ID,
		model.FinishExecutionParams{
			Status:        TaskStatusFailed,
			FinishedAt:    finishedAt,
			ResultSummary: resultSummary,
			ErrorMessage:  errorMessage,
		},
	); err != nil {
		return nil, err
	}

	maxStep, err := l.svcCtx.AuditLogModel.GetMaxStep(l.ctx, tx, taskID)
	if err != nil {
		return nil, err
	}

	message := "task failed by operator: " + operatorID + ", error: " + errorMessage
	if resultSummary != "" {
		message = message + ", result: " + resultSummary
	}

	if _, err = l.svcCtx.AuditLogModel.Insert(l.ctx, tx, &model.AuditLog{
		TaskID:     taskID,
		Step:       maxStep + 1,
		Level:      "error",
		Message:    message,
		ToolName:   "api",
		OccurredAt: finishedAt,
	}); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &types.FailTaskResp{
		Id:     strconv.FormatInt(taskID, 10),
		Status: TaskStatusFailed,
	}, nil
}
