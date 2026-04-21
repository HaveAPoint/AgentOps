package mock

import (
	"context"
	"time"

	"agentops/internal/executor"
)

type Runner struct{}

func NewRunner() *Runner {
	return &Runner{}
}

func (r *Runner) Run(ctx context.Context, req executor.Request) (executor.Result, error) {
	startedAt := time.Now().UTC()

	select {
	case <-ctx.Done():
		return executor.Result{
			StartedAt:  startedAt,
			FinishedAt: time.Now().UTC(),
			Stderr:     ctx.Err().Error(),
		}, ctx.Err()
	default:
	}

	return executor.Result{
		Summary:    "mock executor completed task",
		Stdout:     "mode=" + req.Task.Mode + " repo=" + req.Repo.Path,
		StartedAt:  startedAt,
		FinishedAt: time.Now().UTC(),
	}, nil
}
