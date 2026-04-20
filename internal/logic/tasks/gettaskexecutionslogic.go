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
	"agentops/internal/svc"
	"agentops/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTaskExecutionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTaskExecutionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskExecutionsLogic {
	return &GetTaskExecutionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTaskExecutionsLogic) GetTaskExecutions(req *types.GetTaskExecutionsReq) (resp *types.TaskExecutionsResp, err error) {
	idText := strings.TrimSpace(req.Id)
	if idText == "" {
		return nil, ErrTaskIDRequired
	}

	taskID, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || taskID <= 0 {
		return nil, ErrInvalidTaskID
	}
	actor, err := authctx.CurrentUserFromContext(l.ctx)
	if err != nil {
		return nil, err
	}

	task, err := l.svcCtx.TaskModel.FindByID(l.ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if !canViewTask(actor, *task) {
		return nil, ErrTaskNotFound
	}

	executions, err := l.svcCtx.TaskExecutionModel.ListByTaskID(l.ctx, taskID)
	if err != nil {
		return nil, err
	}

	items := make([]types.ExecutionItem, 0, len(executions))
	for _, execution := range executions {
		startedAt := ""
		if execution.StartedAt.Valid {
			startedAt = execution.StartedAt.Time.UTC().Format(time.RFC3339)
		}

		finishedAt := ""
		if execution.FinishedAt.Valid {
			finishedAt = execution.FinishedAt.Time.UTC().Format(time.RFC3339)
		}

		items = append(items, types.ExecutionItem{
			Id:            strconv.FormatInt(execution.ID, 10),
			TaskId:        strconv.FormatInt(execution.TaskID, 10),
			OperatorId:    execution.OperatorId,
			Status:        execution.Status,
			StartedAt:     startedAt,
			FinishedAt:    finishedAt,
			ResultSummary: execution.ResultSummary,
			ErrorMessage:  execution.ErrorMessage,
		})
	}

	return &types.TaskExecutionsResp{
		Items: items,
	}, nil
}
