package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"agentops/internal/executor"
)

const maxExecutorOutputChars = 2000

type executionResultSummary struct {
	Version               string `json:"version"`
	Input                 summaryInput
	Summary               string   `json:"summary,omitempty"`
	Stdout                string   `json:"stdout,omitempty"`
	Stderr                string   `json:"stderr,omitempty"`
	StartedAt             string   `json:"startedAt,omitempty"`
	FinishedAt            string   `json:"finishedAt,omitempty"`
	GitChangedFilesBefore []string `json:"gitChangedFilesBefore"`
	GitChangedFilesAfter  []string `json:"gitChangedFilesAfter"`
	GitNewChangedFiles    []string `json:"gitNewChangedFiles"`
}

type summaryInput struct {
	TaskID       int64    `json:"taskId"`
	Mode         string   `json:"mode"`
	RepoPath     string   `json:"repoPath"`
	OperatorID   string   `json:"operatorId,omitempty"`
	AllowedPaths []string `json:"allowedPaths"`
	DeniedPaths  []string `json:"deniedPaths"`
	Timeout      string   `json:"timeout"`
}

func buildExecutorResultSummary(prepared *preparedExecution, result executor.Result, beforeFiles []string, afterFiles []string, newFiles []string) string {
	summary := executionResultSummary{
		Version: "v1",
		Input: summaryInput{
			TaskID:       prepared.task.ID,
			Mode:         prepared.task.Mode,
			RepoPath:     prepared.task.RepoPath,
			AllowedPaths: append([]string(nil), prepared.policy.AllowedPaths...),
			DeniedPaths:  append([]string(nil), prepared.policy.DeniedPaths...),
			Timeout:      prepared.timeout.String(),
		},
		Summary:               strings.TrimSpace(result.Summary),
		Stdout:                trimExecutorOutput(result.Stdout),
		Stderr:                trimExecutorOutput(result.Stderr),
		StartedAt:             formatSummaryTime(result.StartedAt),
		FinishedAt:            formatSummaryTime(result.FinishedAt),
		GitChangedFilesBefore: append([]string(nil), beforeFiles...),
		GitChangedFilesAfter:  append([]string(nil), afterFiles...),
		GitNewChangedFiles:    append([]string(nil), newFiles...),
	}

	if prepared.task.OperatorId.Valid {
		summary.Input.OperatorID = prepared.task.OperatorId.String
	}

	encoded, err := json.Marshal(summary)
	if err != nil {
		return "{\"version\":\"v1\",\"summary\":\"serialization_error\"}"
	}

	return string(encoded)
}

func trimExecutorOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}
	if len(output) <= maxExecutorOutputChars {
		return output
	}

	return output[:maxExecutorOutputChars] + "...[truncated]"
}

func buildExecutorErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "executor timeout: " + err.Error()
	case errors.Is(err, context.Canceled):
		return "executor cancelled: " + err.Error()
	default:
		return err.Error()
	}
}

func buildExecutionCancelErrorMessage(reason string, actorRole string, actorID string) string {
	reason = strings.TrimSpace(reason)
	if reason != "" {
		return reason
	}

	actorRole = strings.TrimSpace(actorRole)
	actorID = strings.TrimSpace(actorID)

	if actorRole == "" && actorID == "" {
		return "execution cancelled"
	}
	if actorRole == "" {
		return "execution cancelled by " + actorID
	}
	if actorID == "" {
		return "execution cancelled by " + actorRole
	}

	return "execution cancelled by " + actorRole + ": " + actorID
}

func diffChangedFiles(beforeFiles []string, afterFiles []string) []string {
	beforeSet := make(map[string]struct{}, len(beforeFiles))
	for _, file := range beforeFiles {
		beforeSet[file] = struct{}{}
	}

	diff := make([]string, 0)
	for _, file := range afterFiles {
		if _, ok := beforeSet[file]; ok {
			continue
		}

		diff = append(diff, file)
	}

	return diff
}

func formatSummaryTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}
