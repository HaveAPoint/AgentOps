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

type GetTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskLogic {
	return &GetTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTaskLogic) GetTask(req *types.GetTaskReq) (resp *types.TaskDetailResp, err error) {
	idText := strings.TrimSpace(req.Id)
	if idText == "" {
		return nil, ErrTaskIDRequired
	}

	taskID, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || taskID <= 0 {
		return nil, ErrInvalidTaskID
	}

	task, err := l.svcCtx.TaskModel.FindByID(l.ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	allowedPaths := []string{}
	deniedPaths := []string{}

	policy, err := l.svcCtx.TaskPolicyModel.FindByTaskID(l.ctx, taskID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if policy != nil {
		allowedPaths = policy.AllowedPaths
		deniedPaths = policy.DeniedPaths
	}

	return &types.TaskDetailResp{
		Id:               strconv.FormatInt(task.ID, 10),
		Title:            task.Title,
		RepoPath:         task.RepoPath,
		Prompt:           task.Prompt,
		Mode:             task.Mode,
		Status:           task.Status,
		ApprovalRequired: task.ApprovalRequired,
		MaxSteps:         task.MaxSteps,
		AllowedPaths:     allowedPaths,
		DeniedPaths:      deniedPaths,
		CreatedBy:        task.CreatedBy,
		GitBranch:        task.GitBranch,
		GitHeadCommit:    task.GitHeadCommit,
		GitDirty:         task.GitDirty,
		CreatedAt:        task.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        task.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}
