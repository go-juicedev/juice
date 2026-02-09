package juice

import (
	"context"
	"errors"
	"testing"
)

type managerStub struct {
	object SQLRowsExecutor
	lastV  any
}

func (m *managerStub) Object(v any) SQLRowsExecutor {
	m.lastV = v
	return m.object
}

func TestManagerContextFunctions_manager_test(t *testing.T) {
	ctx := context.Background()

	if _, err := ManagerFromContext(ctx); !errors.Is(err, ErrNoManagerFoundInContext) {
		t.Fatalf("expected ErrNoManagerFoundInContext, got %v", err)
	}

	stub := &managerStub{}
	ctx = ContextWithManager(ctx, stub)

	manager, err := ManagerFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager != stub {
		t.Fatalf("unexpected manager from context")
	}
}

func TestIsTxManager_manager_test(t *testing.T) {
	if IsTxManager(&managerStub{}) {
		t.Fatalf("plain manager should not be tx manager")
	}

	txMgr := &BasicTxManager{basicTxManager: &basicTxManager{}}
	if !IsTxManager(txMgr) {
		t.Fatalf("basic tx manager should be tx manager")
	}
}

func TestNewGenericManager_Object_manager_test(t *testing.T) {
	baseManager := &managerStub{object: &sqlRowsExecutorStub{}}

	gm := NewGenericManager[int](baseManager)
	executor := gm.Object("user")

	if baseManager.lastV != "user" {
		t.Fatalf("expected object arg propagated, got %v", baseManager.lastV)
	}

	if executor == nil {
		t.Fatalf("expected non-nil generic executor")
	}
}

