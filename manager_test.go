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

func TestBasicTxManagerCommitResetsTransaction_manager_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	txManager := &BasicTxManager{
		basicTxManager: &basicTxManager{
			engine: &Engine{db: db},
			ctx:    context.Background(),
		},
	}

	if err := txManager.Begin(); err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if err := txManager.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if txManager.Transaction != nil {
		t.Fatalf("Commit() did not clear transaction")
	}
	if err := txManager.Begin(); err != nil {
		t.Fatalf("Begin() after Commit() error = %v", err)
	}
}

func TestBasicTxManagerRollbackResetsTransaction_manager_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	txManager := &BasicTxManager{
		basicTxManager: &basicTxManager{
			engine: &Engine{db: db},
			ctx:    context.Background(),
		},
	}

	if err := txManager.Begin(); err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	if err := txManager.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if txManager.Transaction != nil {
		t.Fatalf("Rollback() did not clear transaction")
	}
	if err := txManager.Begin(); err != nil {
		t.Fatalf("Begin() after Rollback() error = %v", err)
	}
}
