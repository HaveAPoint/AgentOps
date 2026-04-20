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

	reviewerID := ""
	if task.ReviewerId.Valid {
		reviewerID = task.ReviewerId.String
	}

	operatorID := ""
	if task.OperatorId.Valid {
		operatorID = task.OperatorId.String
	}

	approvedBy := ""
	if task.ApprovedBy.Valid {
		approvedBy = task.ApprovedBy.String
	}

	approvedAt := ""
	if task.ApprovedAt.Valid {
		approvedAt = task.ApprovedAt.Time.UTC().Format(time.RFC3339)
	}

	cancelledBy := ""
	if task.CancelledBy.Valid {
		cancelledBy = task.CancelledBy.String
	}

	cancelledAt := ""
	if task.CancelledAt.Valid {
		cancelledAt = task.CancelledAt.Time.UTC().Format(time.RFC3339)
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
		CreatorId:        task.CreatorId,
		ReviewerId:       reviewerID,
		OperatorId:       operatorID,
		ApprovedBy:       approvedBy,
		ApprovedAt:       approvedAt,
		CancelledBy:      cancelledBy,
		CancelledAt:      cancelledAt,
		GitBranch:        task.GitBranch,
		GitHeadCommit:    task.GitHeadCommit,
		GitDirty:         task.GitDirty,
		CreatedAt:        task.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        task.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil

}
