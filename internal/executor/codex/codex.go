package codex

import (
	"context"

	"agentops/internal/executor"
	"agentops/internal/executor/localcommand"
)

type Runner struct {
	command string
	args    []string
}

func NewRunner(command string, args []string) *Runner {
	return &Runner{
		command: command,
		args:    args,
	}
}

func (r *Runner) Run(ctx context.Context, req executor.Request) (executor.Result, error) {
	return localcommand.Run(ctx, req, localcommand.Options{
		Provider:       "codex",
		Command:        r.command,
		Args:           r.args,
		DefaultCommand: "codex",
		DefaultArgs:    []string{"exec", "-"},
	})
}
