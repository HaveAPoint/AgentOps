package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"agentops/internal/executor"
	"agentops/internal/model"
)

func TestBuildExecutorResultSummaryStructuredJSON(t *testing.T) {
	startedAt := time.Date(2026, 4, 23, 8, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(5 * time.Second)

	prepared := &preparedExecution{
		task: &model.Task{
			ID:       101,
			Mode:     TaskModePatch,
			RepoPath: "/tmp/repo",
			OperatorId: sql.NullString{
				String: "operator-1",
				Valid:  true,
			},
		},
		policy: &model.TaskPolicy{
			AllowedPaths: []string{"internal/logic/tasks"},
			DeniedPaths:  []string{"internal/config"},
		},
		timeout: 30 * time.Second,
	}

	summaryText := buildExecutorResultSummary(
		prepared,
		executor.Result{
			Summary:    "runner completed",
			Stdout:     "runner stdout",
			Stderr:     "runner stderr",
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
		},
		[]string{"README.md"},
		[]string{"README.md", "internal/logic/tasks/new.go"},
		[]string{"internal/logic/tasks/new.go"},
	)

	var summary executionResultSummary
	if err := json.Unmarshal([]byte(summaryText), &summary); err != nil {
		t.Fatalf("summary should be valid json, got err=%v raw=%s", err, summaryText)
	}

	if summary.Version != "v1" {
		t.Fatalf("unexpected version: %q", summary.Version)
	}
	if summary.Input.TaskID != 101 || summary.Input.Mode != TaskModePatch || summary.Input.RepoPath != "/tmp/repo" {
		t.Fatalf("unexpected input snapshot: %+v", summary.Input)
	}
	if summary.Input.OperatorID != "operator-1" {
		t.Fatalf("unexpected operator id: %q", summary.Input.OperatorID)
	}
	if summary.Summary != "runner completed" || summary.Stdout != "runner stdout" || summary.Stderr != "runner stderr" {
		t.Fatalf("unexpected output snapshot: %+v", summary)
	}
	if summary.StartedAt != startedAt.Format(time.RFC3339) || summary.FinishedAt != finishedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected started/finished at: started=%q finished=%q", summary.StartedAt, summary.FinishedAt)
	}
	if len(summary.GitNewChangedFiles) != 1 || summary.GitNewChangedFiles[0] != "internal/logic/tasks/new.go" {
		t.Fatalf("unexpected git new changed files: %+v", summary.GitNewChangedFiles)
	}
}

func TestBuildExecutorResultSummaryTruncatesStdoutAndStderr(t *testing.T) {
	longOutput := strings.Repeat("x", maxExecutorOutputChars+50)

	prepared := &preparedExecution{
		task: &model.Task{
			ID:       102,
			Mode:     TaskModeAnalyze,
			RepoPath: "/tmp/repo",
		},
		policy:  &model.TaskPolicy{},
		timeout: 10 * time.Second,
	}

	summaryText := buildExecutorResultSummary(
		prepared,
		executor.Result{
			Summary: "runner completed",
			Stdout:  longOutput,
			Stderr:  longOutput,
		},
		nil,
		nil,
		nil,
	)

	var summary executionResultSummary
	if err := json.Unmarshal([]byte(summaryText), &summary); err != nil {
		t.Fatalf("summary should be valid json, got err=%v raw=%s", err, summaryText)
	}

	if !strings.HasSuffix(summary.Stdout, "...[truncated]") {
		t.Fatalf("stdout should be truncated, got len=%d", len(summary.Stdout))
	}
	if !strings.HasSuffix(summary.Stderr, "...[truncated]") {
		t.Fatalf("stderr should be truncated, got len=%d", len(summary.Stderr))
	}
}

func TestBuildExecutionCancelErrorMessage(t *testing.T) {
	tests := []struct {
		name      string
		reason    string
		actorRole string
		actorID   string
		want      string
	}{
		{
			name:      "keep explicit reason",
			reason:    "stop running executor",
			actorRole: "operator",
			actorID:   "operator-1",
			want:      "stop running executor",
		},
		{
			name:      "fallback includes role and id",
			reason:    "",
			actorRole: "operator",
			actorID:   "operator-1",
			want:      "execution cancelled by operator: operator-1",
		},
		{
			name:      "fallback role only",
			reason:    "",
			actorRole: "admin",
			actorID:   "",
			want:      "execution cancelled by admin",
		},
		{
			name:      "fallback id only",
			reason:    "",
			actorRole: "",
			actorID:   "operator-1",
			want:      "execution cancelled by operator-1",
		},
		{
			name:      "fallback generic",
			reason:    "",
			actorRole: "",
			actorID:   "",
			want:      "execution cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildExecutionCancelErrorMessage(tt.reason, tt.actorRole, tt.actorID)
			if got != tt.want {
				t.Fatalf("unexpected cancel error message: got=%q want=%q", got, tt.want)
			}
		})
	}
}

func TestBuildExecutorErrorMessage(t *testing.T) {
	timeoutMessage := buildExecutorErrorMessage(context.DeadlineExceeded)
	if timeoutMessage != "executor timeout: context deadline exceeded" {
		t.Fatalf("unexpected timeout message: %q", timeoutMessage)
	}

	cancelMessage := buildExecutorErrorMessage(context.Canceled)
	if cancelMessage != "executor cancelled: context canceled" {
		t.Fatalf("unexpected cancelled message: %q", cancelMessage)
	}

	other := errors.New("runner boom")
	otherMessage := buildExecutorErrorMessage(other)
	if otherMessage != "runner boom" {
		t.Fatalf("unexpected generic message: %q", otherMessage)
	}
}
