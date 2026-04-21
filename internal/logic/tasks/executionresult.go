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
