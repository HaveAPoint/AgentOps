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

	"agentops/internal/svc"
	"agentops/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTaskLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTaskLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskLogsLogic {
	return &GetTaskLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTaskLogsLogic) GetTaskLogs(req *types.GetTaskLogsReq) (resp *types.TaskLogsResp, err error) {
	idText := strings.TrimSpace(req.Id)
	if idText == "" {
		return nil, ErrTaskIDRequired
	}

	taskID, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || taskID <= 0 {
		return nil, ErrInvalidTaskID
	}

	if _, err = l.svcCtx.TaskModel.FindByID(l.ctx, taskID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	logs, err := l.svcCtx.AuditLogModel.ListByTaskID(l.ctx, taskID)
	if err != nil {
		return nil, err
	}

	items := make([]types.LogItem, 0, len(logs))
	for _, log := range logs {
		items = append(items, types.LogItem{
			Step:       log.Step,
			Level:      log.Level,
			Message:    log.Message,
			ToolName:   log.ToolName,
			OccurredAt: log.OccurredAt.UTC().Format(time.RFC3339),
		})
	}

	return &types.TaskLogsResp{
		Items: items,
	}, nil
}
