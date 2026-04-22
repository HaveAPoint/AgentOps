package svc

import (
	"database/sql"
	"fmt"

	"agentops/internal/config"
	"agentops/internal/executor"
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
		TaskRunner:             executormock.NewRunner(),
		ExecutionCancels:       executor.NewCancelRegistry(),
	}
}
