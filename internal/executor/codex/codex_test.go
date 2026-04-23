package codex

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agentops/internal/executor"
)

func TestRunnerUsesDefaultArgsAndStdinPrompt(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "codex-default.sh", `
if [ "$1" != "exec" ] || [ "$2" != "-" ]; then
  echo "unexpected args: $1 $2" >&2
  exit 2
fi
cat
`)

	runner := NewRunner(script, nil)
	result, err := runner.Run(context.Background(), executor.Request{
		Task: executor.TaskInput{
			Prompt: "hello from codex default args",
		},
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	})
	if err != nil {
		t.Fatalf("run codex runner: %v", err)
	}
	if result.Stdout != "hello from codex default args" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestRunnerPrefersExplicitArgs(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "codex-explicit.sh", `
printf "%s" "$1"
`)

	runner := NewRunner(script, []string{"explicit-arg"})
	result, err := runner.Run(context.Background(), executor.Request{
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	})
	if err != nil {
		t.Fatalf("run codex runner: %v", err)
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
