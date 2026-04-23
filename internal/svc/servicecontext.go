package svc

import (
	"database/sql"
	"fmt"
	"strings"

	"agentops/internal/config"
	"agentops/internal/executor"
	executorcodex "agentops/internal/executor/codex"
	executorgemini "agentops/internal/executor/gemini"
	executormock "agentops/internal/executor/mock"
	"agentops/internal/model"
)

type ServiceContext struct {
	Config                 config.Config
	DB                     *sql.DB
	UserModel              *model.UserModel
	TaskModel              *model.TaskModel
	TaskPolicyModel        *model.TaskPolicyModel
	AuditLogModel          *model.AuditLogModel
	ApprovalRecordModel    *model.ApprovalRecordModel
	TaskExecutionModel     *model.TaskExecutionModel
	TaskStatusHistoryModel *model.TaskStatusHistoryModel
	TaskRunner             executor.Runner
	ExecutionCancels       *executor.CancelRegistry
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := model.NewPostgresDB(c.Postgres)

	if err != nil {
		panic(fmt.Sprintf("init postgres failed: %v", err))
	}

	return &ServiceContext{
		Config:                 c,
		DB:                     db,
		UserModel:              model.NewUserModel(db),
		TaskModel:              model.NewTaskModel(db),
		TaskPolicyModel:        model.NewTaskPolicyModel(db),
		AuditLogModel:          model.NewAuditLogModel(db),
		ApprovalRecordModel:    model.NewApprovalRecordModel(db),
		TaskExecutionModel:     model.NewTaskExecutionModel(db),
		TaskStatusHistoryModel: model.NewTaskStatusHistoryModel(db),
		TaskRunner:             newTaskRunner(c.Executor),
		ExecutionCancels:       executor.NewCancelRegistry(),
	}
}

func newTaskRunner(c config.ExecutorConf) executor.Runner {
	switch strings.ToLower(strings.TrimSpace(c.Provider)) {
	case "", "mock":
		return executormock.NewRunner()
	case "codex":
		return executorcodex.NewRunner(c.Command, c.Args)
	case "gemini":
		return executorgemini.NewRunner(c.Command, c.Args)
	default:
		panic(fmt.Sprintf("unsupported executor provider: %s", c.Provider))
	}
}
