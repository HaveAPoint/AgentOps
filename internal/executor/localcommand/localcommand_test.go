package localcommand

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agentops/internal/executor"
)

func TestRunCapturesStdoutStderrAndUsesRepoDir(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "success.sh", `
printf "cwd=%s\n" "$(pwd)"
cat
printf "warn\n" >&2
`)

	result, err := Run(context.Background(), executor.Request{
		Task: executor.TaskInput{
			Prompt: "hello from stdin",
		},
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	}, Options{
		Provider: "test",
		Command:  script,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}

	if !strings.Contains(result.Stdout, "cwd="+repoDir) {
		t.Fatalf("expected stdout to include repo dir, got %q", result.Stdout)
	}
	if !strings.Contains(result.Stdout, "hello from stdin") {
		t.Fatalf("expected stdout to include stdin prompt, got %q", result.Stdout)
	}
	if result.Stderr != "warn" {
		t.Fatalf("expected stderr warning, got %q", result.Stderr)
	}
	if result.Summary != "test executor command completed" {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
}

func TestRunReturnsErrorAndCapturedStderr(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "fail.sh", `
printf "bad\n" >&2
exit 7
`)

	result, err := Run(context.Background(), executor.Request{
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	}, Options{
		Provider: "test",
		Command:  script,
	})
	if err == nil {
		t.Fatal("expected command error")
	}
	if result.Summary != "test executor command failed" {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
	if result.Stderr != "bad" {
		t.Fatalf("expected captured stderr, got %q", result.Stderr)
	}
}

func TestRunUsesDefaultArgsWhenArgsEmpty(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "args.sh", `
printf "%s" "$1"
`)

	result, err := Run(context.Background(), executor.Request{
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	}, Options{
		Provider:    "test",
		Command:     script,
		DefaultArgs: []string{"default-arg"},
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if result.Stdout != "default-arg" {
		t.Fatalf("expected default arg stdout, got %q", result.Stdout)
	}
}

func TestRunPrefersExplicitArgsOverDefaultArgs(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "args.sh", `
printf "%s" "$1"
`)

	result, err := Run(context.Background(), executor.Request{
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	}, Options{
		Provider:    "test",
		Command:     script,
		Args:        []string{"explicit-arg"},
		DefaultArgs: []string{"default-arg"},
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if result.Stdout != "explicit-arg" {
		t.Fatalf("expected explicit arg stdout, got %q", result.Stdout)
	}
}

func TestRunReplacesPromptPlaceholderInArgs(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "prompt.sh", `
printf "%s" "$1"
`)

	prompt := "prompt-via-arg"
	result, err := Run(context.Background(), executor.Request{
		Task: executor.TaskInput{
			Prompt: prompt,
		},
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	}, Options{
		Provider: "test",
		Command:  script,
		Args:     []string{PromptArgPlaceholder},
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if result.Stdout != prompt {
		t.Fatalf("expected prompt placeholder to be replaced, got %q", result.Stdout)
	}
}

func TestRunReplacesPromptPlaceholderInDefaultArgsWithoutMutatingOptions(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "prompt-default.sh", `
printf "%s" "$2"
`)

	defaultArgs := []string{"-p", PromptArgPlaceholder}
	prompt := "prompt-from-default-args"

	result, err := Run(context.Background(), executor.Request{
		Task: executor.TaskInput{
			Prompt: prompt,
		},
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	}, Options{
		Provider:    "test",
		Command:     script,
		DefaultArgs: defaultArgs,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if result.Stdout != prompt {
		t.Fatalf("expected prompt placeholder in default args to be replaced, got %q", result.Stdout)
	}
	if defaultArgs[1] != PromptArgPlaceholder {
		t.Fatalf("expected default args to remain unchanged, got %q", defaultArgs[1])
	}
}

func TestRunHonorsContextCancellation(t *testing.T) {
	repoDir := t.TempDir()
	script := writeShellScript(t, repoDir, "sleep.sh", `
sleep 5
`)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := Run(ctx, executor.Request{
		Repo: executor.RepoContext{
			Path: repoDir,
		},
	}, Options{
		Provider: "test",
		Command:  script,
	})
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got err=%v ctxErr=%v", err, ctx.Err())
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
