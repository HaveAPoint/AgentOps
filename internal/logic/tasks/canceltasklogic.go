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

type CancelTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelTaskLogic {
	return &CancelTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CancelTaskLogic) CancelTask(req *types.CancelTaskReq) (resp *types.CancelTaskResp, err error) {
	idText := strings.TrimSpace(req.Id)
	if idText == "" {
		return nil, ErrTaskIDRequired
	}

	actor, err := authctx.CurrentUserFromContext(l.ctx)
	if err != nil {
		return nil, err
	}

	actorID := actor.ID
	reason := strings.TrimSpace(req.Reason)

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

	switch task.Status {
	case TaskStatusPending, TaskStatusWaitingApproval, TaskStatusRunning:
	default:
		return nil, ErrTaskCannotBeCancelled
	}

	actorRole, err := cancelActorRole(actor, task)
	if err != nil {
		return nil, err
	}

	cancelledAt := time.Now().UTC()

	var runningExecution *model.TaskExecution
	if task.Status == TaskStatusRunning {
		runningExecution, err = l.svcCtx.TaskExecutionModel.FindLatestRunningByTaskIDForUpdate(l.ctx, tx, taskID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrRunningExecutionNotFound
			}
			return nil, err
		}

		if err = l.svcCtx.TaskExecutionModel.Finish(
			l.ctx,
			tx,
			runningExecution.ID,
			model.FinishExecutionParams{
				Status:        TaskStatusCancelled,
				FinishedAt:    cancelledAt,
				ResultSummary: "",
				ErrorMessage:  reason,
			},
		); err != nil {
			return nil, err
		}
	}

	if _, err = l.svcCtx.TaskModel.Cancel(l.ctx, tx, taskID, actorID, cancelledAt); err != nil {
		return nil, err
	}

	if _, err = l.svcCtx.TaskStatusHistoryModel.Insert(l.ctx, tx, &model.TaskStatusHistory{
		TaskID: taskID,
		FromStatus: sql.NullString{
			String: task.Status,
			Valid:  task.Status != "",
		},
		ToStatus:  TaskStatusCancelled,
		Action:    "cancel",
		ActorID:   actorID,
		ActorRole: actorRole,
		Reason:    reason,
	}); err != nil {
		return nil, err
	}

	maxStep, err := l.svcCtx.AuditLogModel.GetMaxStep(l.ctx, tx, taskID)
	if err != nil {
		return nil, err
	}

	message := "task cancelled by " + actorRole + ": " + actorID
	if reason != "" {
		message = message + ", reason: " + reason
	}

	if _, err = l.svcCtx.AuditLogModel.Insert(l.ctx, tx, &model.AuditLog{
		TaskID:     taskID,
		Step:       maxStep + 1,
		Level:      "info",
		Message:    message,
		ToolName:   "api",
		OccurredAt: cancelledAt,
	}); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	if task.Status == TaskStatusRunning {
		l.svcCtx.ExecutionCancels.Cancel(taskID)
	}

	return &types.CancelTaskResp{
		Id:     strconv.FormatInt(taskID, 10),
		Status: TaskStatusCancelled,
	}, nil
}

func cancelActorRole(actor authctx.CurrentUser, task *model.Task) (string, error) {
	if actor.SystemRole == authctx.SystemRoleAdmin {
		return "admin", nil
	}

	switch task.Status {
	case TaskStatusWaitingApproval, TaskStatusPending:
		switch {
		case task.CreatorId == actor.ID:
			return "creator", nil
		case actor.SystemRole == authctx.SystemRoleReviewer &&
			task.ReviewerId.Valid &&
			task.ReviewerId.String == actor.ID:
			return "reviewer", nil
		default:
			return "", ErrPermissionDenied
		}

	case TaskStatusRunning:
		if actor.SystemRole == authctx.SystemRoleOperator &&
			task.OperatorId.Valid &&
			task.OperatorId.String == actor.ID {
			return "operator", nil
		}
		return "", ErrPermissionDenied

	default:
		return "", ErrTaskCannotBeCancelled
	}
}
