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

type GetTaskStatusHistoriesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTaskStatusHistoriesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskStatusHistoriesLogic {
	return &GetTaskStatusHistoriesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTaskStatusHistoriesLogic) GetTaskStatusHistories(req *types.GetTaskStatusHistoriesReq) (resp *types.TaskStatusHistoriesResp, err error) {
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

	histories, err := l.svcCtx.TaskStatusHistoryModel.ListByTaskID(l.ctx, taskID)
	if err != nil {
		return nil, err
	}

	items := make([]types.StatusHistoryItem, 0, len(histories))
	for _, history := range histories {
		fromStatus := ""
		if history.FromStatus.Valid {
			fromStatus = history.FromStatus.String
		}

		items = append(items, types.StatusHistoryItem{
			Id:         strconv.FormatInt(history.ID, 10),
			FromStatus: fromStatus,
			ToStatus:   history.ToStatus,
			Action:     history.Action,
			ActorId:    history.ActorID,
			ActorRole:  history.ActorRole,
			Reason:     history.Reason,
			CreatedAt:  history.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	return &types.TaskStatusHistoriesResp{
		Items: items,
	}, nil
}
