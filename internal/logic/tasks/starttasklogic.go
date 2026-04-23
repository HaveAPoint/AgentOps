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
	"agentops/internal/executor"
	"agentops/internal/gitctx"
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

type preparedExecution struct {
	task               *model.Task
	policy             *model.TaskPolicy
	timeout            time.Duration
	changedFilesBefore []string
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

	prepared, err := l.prepareExecution(taskID, operatorID)
	if err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithTimeout(l.ctx, prepared.timeout)
	l.svcCtx.ExecutionCancels.Register(taskID, cancel)
	defer l.svcCtx.ExecutionCancels.Unregister(taskID)
	defer cancel()

	result, runErr := l.svcCtx.TaskRunner.Run(runCtx, buildExecutorRequest(prepared.task, actor, prepared.policy, prepared.timeout))
	resultSummary, runErr := inspectExecutionResult(prepared, result, runErr)

	if runErr != nil {
		return l.handleExecutionRunError(taskID, resultSummary, runErr)
	}

	return l.handleExecutionSucceed(taskID, resultSummary)
}

func (l *StartTaskLogic) prepareExecution(taskID int64, operatorID string) (*preparedExecution, error) {
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

	if !task.OperatorId.Valid || task.OperatorId.String != operatorID {
		return nil, ErrPermissionDenied
	}

	if err = validateExecutionRepo(task.RepoPath, l.svcCtx.Config.Executor.AllowedRepoPaths); err != nil {
		return nil, err
	}

	policy, err := l.svcCtx.TaskPolicyModel.FindByTaskID(l.ctx, taskID)
	if err != nil {
		return nil, err
	}

	if err = validateExecutionPolicy(task, policy); err != nil {
		return nil, err
	}

	executionTimeout := l.svcCtx.Config.Executor.Timeout()
	now := time.Now().UTC()

	changedFilesBefore, err := gitctx.ChangedFiles(task.RepoPath)
	if err != nil {
		return nil, err
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
		Message:    buildExecutionStartMessage(operatorID, task, policy, executionTimeout),
		ToolName:   "api",
		OccurredAt: now,
	}); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &preparedExecution{
		task:               task,
		policy:             policy,
		timeout:            executionTimeout,
		changedFilesBefore: changedFilesBefore,
	}, nil
}

func buildExecutionStartMessage(operatorID string, task *model.Task, policy *model.TaskPolicy, timeout time.Duration) string {
	return "task started by operator: " + operatorID +
		", repo: " + task.RepoPath +
		", mode: " + task.Mode +
		", allowedPaths: [" + strings.Join(policy.AllowedPaths, ",") + "]" +
		", deniedPaths: [" + strings.Join(policy.DeniedPaths, ",") + "]" +
		", timeout: " + timeout.String()
}

func buildExecutorRequest(task *model.Task, actor authctx.CurrentUser, policy *model.TaskPolicy, timeout time.Duration) executor.Request {
	return executor.Request{
		Task: executor.TaskInput{
			ID:       task.ID,
			Title:    task.Title,
			Prompt:   task.Prompt,
			Mode:     task.Mode,
			MaxSteps: task.MaxSteps,
		},
		Operator: actor,
		Repo: executor.RepoContext{
			Path:       task.RepoPath,
			Branch:     task.GitBranch,
			HeadCommit: task.GitHeadCommit,
			Dirty:      task.GitDirty,
		},
		Policy: executor.PolicyContext{
			AllowedPaths: policy.AllowedPaths,
			DeniedPaths:  policy.DeniedPaths,
		},
		Timeout: timeout,
	}
}

func inspectExecutionResult(prepared *preparedExecution, result executor.Result, runErr error) (string, error) {
	changedFilesAfter, inspectErr := gitctx.ChangedFiles(prepared.task.RepoPath)
	newChangedFiles := diffChangedFiles(prepared.changedFilesBefore, changedFilesAfter)

	resultSummary := buildExecutorResultSummary(prepared, result, prepared.changedFilesBefore, changedFilesAfter, newChangedFiles)

	if inspectErr != nil && runErr == nil {
		runErr = inspectErr
	}

	if runErr == nil {
		if err := validateChangedFilesAllowed(prepared.task.Mode, prepared.policy, newChangedFiles); err != nil {
			runErr = err
		}
	}

	return resultSummary, runErr
}

func (l *StartTaskLogic) handleExecutionRunError(taskID int64, resultSummary string, runErr error) (*types.StartTaskResp, error) {
	id := strconv.FormatInt(taskID, 10)

	if errors.Is(runErr, context.Canceled) {
		cancelledTask, findErr := l.svcCtx.TaskModel.FindByID(l.ctx, taskID)
		if findErr != nil {
			return nil, findErr
		}
		if cancelledTask.Status == TaskStatusCancelled {
			return &types.StartTaskResp{
				Id:     id,
				Status: TaskStatusCancelled,
			}, nil
		}
	}

	failLogic := NewFailTaskLogic(l.ctx, l.svcCtx)
	if _, err := failLogic.FailTask(&types.FailTaskReq{
		Id:            id,
		ResultSummary: resultSummary,
		ErrorMessage:  buildExecutorErrorMessage(runErr),
	}); err != nil {
		return nil, err
	}

	return &types.StartTaskResp{
		Id:     id,
		Status: TaskStatusFailed,
	}, nil
}

func (l *StartTaskLogic) handleExecutionSucceed(taskID int64, resultSummary string) (*types.StartTaskResp, error) {
	id := strconv.FormatInt(taskID, 10)

	succeedLogic := NewSucceedTaskLogic(l.ctx, l.svcCtx)
	if _, err := succeedLogic.SucceedTask(&types.SucceedTaskReq{
		Id:            id,
		ResultSummary: resultSummary,
	}); err != nil {
		if errors.Is(err, ErrTaskNotRunning) {
			currentTask, findErr := l.svcCtx.TaskModel.FindByID(l.ctx, taskID)
			if findErr != nil {
				return nil, findErr
			}

			switch currentTask.Status {
			case TaskStatusCancelled, TaskStatusFailed, TaskStatusSucceeded:
				return &types.StartTaskResp{
					Id:     id,
					Status: currentTask.Status,
				}, nil
			}
		}

		return nil, err
	}

	return &types.StartTaskResp{
		Id:     id,
		Status: TaskStatusSucceeded,
	}, nil
}
