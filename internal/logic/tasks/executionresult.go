package tasks

import (
	"context"
	"errors"
	"strings"

	"agentops/internal/executor"
)

const maxExecutorOutputChars = 2000

func buildExecutorResultSummary(result executor.Result) string {
	parts := make([]string, 0, 3)

	if strings.TrimSpace(result.Summary) != "" {
		parts = append(parts, "summary: "+strings.TrimSpace(result.Summary))
	}

	if strings.TrimSpace(result.Stdout) != "" {
		parts = append(parts, "stdout: "+trimExecutorOutput(result.Stdout))
	}

	if strings.TrimSpace(result.Stderr) != "" {
		parts = append(parts, "stderr: "+trimExecutorOutput(result.Stderr))
	}

	return strings.Join(parts, "\n")
}

func trimExecutorOutput(output string) string {
	output = strings.TrimSpace(output)
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

func appendGitChangedFilesSummary(summary string, beforeFiles []string, afterFiles []string, newFiles []string) string {
	parts := make([]string, 0, 3)

	if strings.TrimSpace(summary) != "" {
		parts = append(parts, summary)
	}

	parts = append(parts, "gitChangedFilesBefore: ["+strings.Join(beforeFiles, ",")+"]")
	parts = append(parts, "gitChangedFilesAfter: ["+strings.Join(afterFiles, ",")+"]")
	parts = append(parts, "gitNewChangedFiles: ["+strings.Join(newFiles, ",")+"]")

	return strings.Join(parts, "\n")
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
