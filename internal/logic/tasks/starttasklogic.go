// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package tasks

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"

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

	operatorID := strings.TrimSpace(req.OperatorId)
	if operatorID == "" {
		return nil, ErrOperatorIDRequired
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

	if task.Status != TaskStatusPending {
		return nil, ErrTaskNotPending
	}

	if _, err = l.svcCtx.TaskModel.UpdateStatus(l.ctx, tx, taskID, TaskStatusRunning); err != nil {
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

