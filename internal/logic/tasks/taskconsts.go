package tasks

const (
	TaskModeAnalyze = "analyze"
	TaskModePatch   = "patch"
)

const (
	TaskStatusPending         = "pending"
	TaskStatusWaitingApproval = "waiting_approval"
	TaskStatusRunning         = "running"
	TaskStatusSucceeded       = "succeeded"
	TaskStatusFailed          = "failed"
	TaskStatusCancelled       = "cancelled"
)
