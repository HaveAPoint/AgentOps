package svc

import (
	"testing"

	"agentops/internal/config"
	executorcodex "agentops/internal/executor/codex"
	executorgemini "agentops/internal/executor/gemini"
	executormock "agentops/internal/executor/mock"
)

func TestNewTaskRunnerDefaultsToMock(t *testing.T) {
	runner := newTaskRunner(config.ExecutorConf{})

	if _, ok := runner.(*executormock.Runner); !ok {
		t.Fatalf("expected mock runner, got %T", runner)
	}
}

func TestNewTaskRunnerSupportsMockProvider(t *testing.T) {
	runner := newTaskRunner(config.ExecutorConf{
		Provider: " mock ",
	})

	if _, ok := runner.(*executormock.Runner); !ok {
		t.Fatalf("expected mock runner, got %T", runner)
	}
}

func TestNewTaskRunnerSupportsCodexProvider(t *testing.T) {
	runner := newTaskRunner(config.ExecutorConf{
		Provider: "codex",
	})

	if _, ok := runner.(*executorcodex.Runner); !ok {
		t.Fatalf("expected codex runner, got %T", runner)
	}
}

func TestNewTaskRunnerSupportsGeminiProvider(t *testing.T) {
	runner := newTaskRunner(config.ExecutorConf{
		Provider: "gemini",
	})

	if _, ok := runner.(*executorgemini.Runner); !ok {
		t.Fatalf("expected gemini runner, got %T", runner)
	}
}

func TestNewTaskRunnerRejectsUnknownProvider(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected unsupported provider panic")
		}
	}()

	_ = newTaskRunner(config.ExecutorConf{
		Provider: "unknown",
	})
}
