package tasks

import (
	"errors"
	"testing"

	"agentops/internal/model"
)

func TestValidateExecutionPolicy(t *testing.T) {
	tests := []struct {
		name    string
		task    *model.Task
		policy  *model.TaskPolicy
		wantErr error
	}{
		{
			name: "analyze allows empty path policy",
			task: &model.Task{
				Mode: TaskModeAnalyze,
			},
			policy: &model.TaskPolicy{},
		},
		{
			name: "patch requires allowed paths",
			task: &model.Task{
				Mode: TaskModePatch,
			},
			policy:  &model.TaskPolicy{},
			wantErr: ErrAllowedPathRequiredForPatch,
		},
		{
			name: "patch accepts repo relative path",
			task: &model.Task{
				Mode: TaskModePatch,
			},
			policy: &model.TaskPolicy{
				AllowedPaths: []string{"internal/logic/tasks"},
			},
		},
		{
			name: "analyze rejects unsafe declared path",
			task: &model.Task{
				Mode: TaskModeAnalyze,
			},
			policy: &model.TaskPolicy{
				AllowedPaths: []string{"../secret"},
			},
			wantErr: ErrInvalidPolicyPath,
		},
		{
			name: "unknown mode is rejected",
			task: &model.Task{
				Mode: "unknown",
			},
			policy:  &model.TaskPolicy{},
			wantErr: ErrInvalidMode,
		},
		{
			name: "policy is required",
			task: &model.Task{
				Mode: TaskModeAnalyze,
			},
			policy:  nil,
			wantErr: ErrExecutionPolicyRequired,
		},
		{
			name: "absolute path is rejected",
			task: &model.Task{
				Mode: TaskModePatch,
			},
			policy: &model.TaskPolicy{
				AllowedPaths: []string{"/tmp/repo"},
			},
			wantErr: ErrInvalidPolicyPath,
		},
		{
			name: "parent traversal is rejected",
			task: &model.Task{
				Mode: TaskModePatch,
			},
			policy: &model.TaskPolicy{
				AllowedPaths: []string{"../secret"},
			},
			wantErr: ErrInvalidPolicyPath,
		},
		{
			name: "repo root is rejected",
			task: &model.Task{
				Mode: TaskModePatch,
			},
			policy: &model.TaskPolicy{
				AllowedPaths: []string{"."},
			},
			wantErr: ErrInvalidPolicyPath,
		},
		{
			name: "windows separator is rejected",
			task: &model.Task{
				Mode: TaskModePatch,
			},
			policy: &model.TaskPolicy{
				AllowedPaths: []string{`internal\logic`},
			},
			wantErr: ErrInvalidPolicyPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExecutionPolicy(tt.task, tt.policy)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}
