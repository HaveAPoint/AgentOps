package localcommand

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"agentops/internal/executor"
)

type Options struct {
	Provider       string
	Command        string
	Args           []string
	DefaultCommand string
	DefaultArgs    []string
}

const PromptArgPlaceholder = "{TASK_PROMPT}"

func Run(ctx context.Context, req executor.Request, opts Options) (executor.Result, error) {
	startedAt := time.Now().UTC()

	command := strings.TrimSpace(opts.Command)
	if command == "" {
		command = opts.DefaultCommand
	}

	args := opts.Args
	if len(args) == 0 {
		args = opts.DefaultArgs
	}
	args = replacePromptArgPlaceholder(args, req.Task.Prompt)

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = req.Repo.Path
	cmd.Stdin = strings.NewReader(req.Task.Prompt)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	finishedAt := time.Now().UTC()

	result := executor.Result{
		Summary:    fmt.Sprintf("%s executor command completed", opts.Provider),
		Stdout:     strings.TrimSpace(stdout.String()),
		Stderr:     strings.TrimSpace(stderr.String()),
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}
	if err != nil {
		result.Summary = fmt.Sprintf("%s executor command failed", opts.Provider)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return result, fmt.Errorf("%s command %q interrupted: %w", opts.Provider, command, ctxErr)
		}
		return result, fmt.Errorf("%s command %q failed: %w", opts.Provider, command, err)
	}

	return result, nil
}

func replacePromptArgPlaceholder(args []string, prompt string) []string {
	if len(args) == 0 {
		return nil
	}

	resolved := make([]string, len(args))
	copy(resolved, args)

	for i, arg := range resolved {
		if arg == PromptArgPlaceholder {
			resolved[i] = prompt
		}
	}

	return resolved
}
