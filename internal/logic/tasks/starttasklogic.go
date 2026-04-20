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

	authctx "agentops/internal/auth"
	"agentops/internal/model"
	"agentops/internal/svc"
	"agentops/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStartTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartTaskLogic {
	return &StartTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StartTaskLogic) StartTask(req *types.StartTaskReq) (resp *types.StartTaskResp, err error) {
	idText := strings.TrimSpace(req.Id)
	if idText == "" {
		return nil, ErrTaskIDRequired
	}

	actor, err := authctx.CurrentUserFromContext(l.ctx)
	if err != nil {
		return nil, err
	}
	if actor.SystemRole != authctx.SystemRoleOperator {
		return nil, ErrPermissionDenied
	}

	operatorID := actor.ID

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

	if task.Status != TaskStatusPending {
		return nil, ErrTaskNotPending
	}

	now := time.Now().UTC()

	if !task.OperatorId.Valid || task.OperatorId.String != operatorID {
		return nil, ErrPermissionDenied
	}

	if _, err = l.svcCtx.TaskModel.Start(l.ctx, tx, taskID, operatorID); err != nil {
		return nil, err
	}

	if _, err = l.svcCtx.TaskStatusHistoryModel.Insert(l.ctx, tx, &model.TaskStatusHistory{
		TaskID: taskID,
		FromStatus: sql.NullString{
			String: task.Status,
			Valid:  task.Status != "",
		},
		ToStatus:  TaskStatusRunning,
		Action:    "start",
		ActorID:   operatorID,
		ActorRole: "operator",
		Reason:    "",
	}); err != nil {
		return nil, err
	}

	if _, err = l.svcCtx.TaskExecutionModel.Insert(l.ctx, tx, &model.TaskExecution{
		TaskID:     taskID,
		OperatorId: operatorID,
		Status:     TaskStatusRunning,
		StartedAt: sql.NullTime{
			Time:  now,
			Valid: true,
		},
		ResultSummary: "",
		ErrorMessage:  "",
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
		Message:    "task started by operator: " + operatorID,
		ToolName:   "api",
		OccurredAt: now,
	}); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &types.StartTaskResp{
		Id:     strconv.FormatInt(taskID, 10),
		Status: TaskStatusRunning,
	}, nil
}
