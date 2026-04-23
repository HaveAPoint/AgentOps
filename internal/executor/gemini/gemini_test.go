package gemini

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentops/internal/executor"
)

func TestRunnerUsesDefaultPromptArg(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "gemini-default.sh", `
if [ "$1" != "-p" ]; then
  echo "unexpected arg1: $1" >&2
  exit 2
fi
printf "%s" "$2"
`)

	runner := NewRunner(script, nil)
	result, err := runner.Run(context.Background(), executor.Request{
		Task: executor.TaskInput{
			Prompt: "hello from gemini prompt arg",
		},
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	})
	if err != nil {
		t.Fatalf("run gemini runner: %v", err)
	}
	if result.Stdout != "hello from gemini prompt arg" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestRunnerPrefersExplicitArgs(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "gemini-explicit.sh", `
printf "%s" "$1"
`)

	runner := NewRunner(script, []string{"explicit-arg"})
	result, err := runner.Run(context.Background(), executor.Request{
		Task: executor.TaskInput{
			Prompt: "should-not-be-used",
		},
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	})
	if err != nil {
		t.Fatalf("run gemini runner: %v", err)
	}
	if result.Stdout != "explicit-arg" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func writeShellScript(t *testing.T, dir string, name string, body string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nset -eu\n" + strings.TrimSpace(body) + "\n"
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	return path
}
