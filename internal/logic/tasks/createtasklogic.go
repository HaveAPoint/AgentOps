package tasks

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	authctx "agentops/internal/auth"
	"agentops/internal/gitctx"
	"agentops/internal/model"
	"agentops/internal/svc"
	"agentops/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateTaskLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTaskLogic {
	return &CreateTaskLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateTaskLogic) CreateTask(req *types.CreateTaskReq) (resp *types.CreateTaskResp, err error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrTitleRequired
	}

	repoPath := strings.TrimSpace(req.RepoPath)
	if repoPath == "" {
		return nil, ErrRepoPathRequired
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, ErrPromptRequired
	}

	creator, err := authctx.CurrentUserFromContext(l.ctx)
	if err != nil {
		return nil, err
	}
	creatorID := creator.ID

	reviewerID := strings.TrimSpace(req.ReviewerId)
	operatorID := strings.TrimSpace(req.OperatorId)

	if req.Mode != TaskModeAnalyze && req.Mode != TaskModePatch {
		return nil, ErrInvalidMode
	}

	if req.MaxSteps <= 0 {
		return nil, ErrInvalidMaxSteps
	}

	repoInfo, err := gitctx.Read(repoPath)
	if err != nil {
		if errors.Is(err, gitctx.ErrNotGitRepo) {
			return nil, ErrRepoNotGitRepo
		}
		if errors.Is(err, gitctx.ErrNoCommitYet) {
			return nil, ErrRepoHasNoCommits
		}
		return nil, err
	}

	status := TaskStatusPending
	if req.ApprovalRequired {
		status = TaskStatusWaitingApproval
	}

	tx, err := l.svcCtx.DB.BeginTx(l.ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	taskID, err := l.svcCtx.TaskModel.Insert(l.ctx, tx, &model.Task{
		Title:            title,
		RepoPath:         repoPath,
		Prompt:           prompt,
		Mode:             req.Mode,
		Status:           status,
		ApprovalRequired: req.ApprovalRequired,
		MaxSteps:         req.MaxSteps,
		CreatorId:        creatorID,
		ReviewerId: sql.NullString{
			String: reviewerID,
			Valid:  reviewerID != "",
		},
		OperatorId: sql.NullString{
			String: operatorID,
			Valid:  operatorID != "",
		},
		GitBranch:     repoInfo.Branch,
		GitHeadCommit: repoInfo.HeadCommit,
		GitDirty:      repoInfo.Dirty,
	})

	if err != nil {
		return nil, err
	}

	_, err = l.svcCtx.TaskPolicyModel.Insert(l.ctx, tx, &model.TaskPolicy{
		TaskID:       taskID,
		AllowedPaths: req.AllowedPaths,
		DeniedPaths:  req.DeniedPaths,
	})
	if err != nil {
		return nil, err
	}

	if _, err = l.svcCtx.TaskStatusHistoryModel.Insert(l.ctx, tx, &model.TaskStatusHistory{
		TaskID:     taskID,
		FromStatus: sql.NullString{},
		ToStatus:   status,
		Action:     "create",
		ActorID:    creatorID,
		ActorRole:  "creator",
		Reason:     "",
	}); err != nil {
		return nil, err
	}

	_, err = l.svcCtx.AuditLogModel.Insert(l.ctx, tx, &model.AuditLog{
		TaskID:     taskID,
		Step:       1,
		Level:      "info",
		Message:    "task created by creator: " + creatorID,
		ToolName:   "api",
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &types.CreateTaskResp{
		Id:     strconv.FormatInt(taskID, 10),
		Status: status,
	}, nil
}
