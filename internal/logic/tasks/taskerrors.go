package tasks

import "errors"

var (
	ErrTaskIDRequired         = errors.New("id is required")
	ErrInvalidTaskID          = errors.New("id must be a numeric task id")
	ErrTaskNotFound           = errors.New("task not found")
	ErrTitleRequired          = errors.New("title is required")
	ErrRepoPathRequired       = errors.New("repoPath is required")
	ErrRepoNotGitRepo         = errors.New("repoPath must point to a git repository")
	ErrPromptRequired         = errors.New("prompt is required")
	ErrInvalidMode            = errors.New("mode must be analyze or patch")
	ErrInvalidMaxSteps        = errors.New("maxSteps must be greater than 0")
	ErrTaskNotWaitingApproval = errors.New("task is not waiting for approval")
	ErrTaskCannotBeCancelled  = errors.New("task cannot be cancelled from current status")
	ErrRepoHasNoCommits       = errors.New("git repository has no commits yet")
)
