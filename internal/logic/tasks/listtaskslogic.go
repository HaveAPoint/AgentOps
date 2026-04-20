// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package tasks

import (
	"context"
	"strconv"
	"time"

	authctx "agentops/internal/auth"
	"agentops/internal/model"
	"agentops/internal/svc"
	"agentops/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTasksLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTasksLogic {
	return &ListTasksLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTasksLogic) ListTasks() (resp *types.TaskListResp, err error) {
	actor, err := authctx.CurrentUserFromContext(l.ctx)
	if err != nil {
		return nil, err
	}

	tasks, err := l.svcCtx.TaskModel.List(l.ctx)
	if err != nil {
		return nil, err
	}

	visibleTasks := make([]model.Task, 0, len(tasks))
	for _, task := range tasks {
		if canViewTask(actor, task) {
			visibleTasks = append(visibleTasks, task)
		}
	}

	items := make([]types.TaskItem, 0, len(visibleTasks))

	for _, task := range visibleTasks {
		items = append(items, types.TaskItem{
			Id:               strconv.FormatInt(task.ID, 10),
			Title:            task.Title,
			RepoPath:         task.RepoPath,
			Mode:             task.Mode,
			Status:           task.Status,
			ApprovalRequired: task.ApprovalRequired,
			CreatorId:        task.CreatorId,
			GitBranch:        task.GitBranch,
			GitHeadCommit:    task.GitHeadCommit,
			GitDirty:         task.GitDirty,
			CreatedAt:        task.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:        task.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return &types.TaskListResp{
		Items: items,
	}, nil
}

func canViewTask(actor authctx.CurrentUser, task model.Task) bool {
	if actor.SystemRole == authctx.SystemRoleAdmin {
		return true
	}

	if task.CreatorId == actor.ID {
		return true
	}

	if task.ReviewerId.Valid && task.ReviewerId.String == actor.ID {
		return true
	}

	if task.OperatorId.Valid && task.OperatorId.String == actor.ID {
		return true
	}

	return false
}
