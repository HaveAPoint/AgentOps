package executor

import (
	"context"
	"errors"
	"time"

	authctx "agentops/internal/auth"
)

var ErrProviderNotImplemented = errors.New("executor provider not implemented")

type TaskInput struct {
	ID       int64
	Title    string
	Prompt   string
	Mode     string
	MaxSteps int64
}

type RepoContext struct {
	Path       string
	Branch     string
	HeadCommit string
	Dirty      bool
}

type PolicyContext struct {
	AllowedPaths []string
	DeniedPaths  []string
}

type Request struct {
	Task     TaskInput
	Operator authctx.CurrentUser
	Repo     RepoContext
	Policy   PolicyContext
	Timeout  time.Duration
}

type Result struct {
	Summary    string
	Stdout     string
	Stderr     string
	StartedAt  time.Time
	FinishedAt time.Time
}

type Runner interface {
	Run(ctx context.Context, req Request) (Result, error)
}
