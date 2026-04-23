package executor

import (
	"context"
	"testing"
)

func TestCancelRegistryCancelRegisteredTask(t *testing.T) {
	registry := NewCancelRegistry()

	ctx, cancel := context.WithCancel(context.Background())
	registry.Register(123, cancel)

	if ok := registry.Cancel(123); !ok {
		t.Fatal("expected registered task to be cancelled")
	}

	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected context to be cancelled")
	}

	if err := ctx.Err(); err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCancelRegistryCancelMissingTask(t *testing.T) {
	registry := NewCancelRegistry()

	if ok := registry.Cancel(404); ok {
		t.Fatal("expected missing task cancel to return false")
	}
}

func TestCancelRegistryUnregister(t *testing.T) {
	registry := NewCancelRegistry()

	_, cancel := context.WithCancel(context.Background())
	registry.Register(123, cancel)
	registry.Unregister(123)

	if ok := registry.Cancel(123); ok {
		t.Fatal("expected unregistered task cancel to return false")
	}
}
