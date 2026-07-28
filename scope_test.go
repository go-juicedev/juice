package juice

import (
	"context"
	"errors"
	"testing"

	"github.com/go-juicedev/juice/session/tx"
)

func TestTransactionRejectsInvalidManager_scope_test(t *testing.T) {
	handler := func(context.Context, Manager) error { return nil }

	if err := Transaction(context.Background(), nil, handler); !errors.Is(err, ErrInvalidManager) {
		t.Fatalf("Transaction(nil) error = %v, want %v", err, ErrInvalidManager)
	}
	if err := Transaction(context.Background(), &managerStub{}, handler); !errors.Is(err, ErrInvalidManager) {
		t.Fatalf("Transaction(managerStub) error = %v, want %v", err, ErrInvalidManager)
	}
	manualManager := &BasicTxManager{basicTxManager: &basicTxManager{}}
	if err := Transaction(context.Background(), manualManager, handler); !errors.Is(err, ErrInvalidManager) {
		t.Fatalf("Transaction(BasicTxManager) error = %v, want %v", err, ErrInvalidManager)
	}
}

func TestTransactionStartsTransactionForEngine_scope_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	engine := &Engine{db: db}

	handlerCalled := false
	err := Transaction(context.Background(), engine, func(_ context.Context, manager Manager) error {
		handlerCalled = true
		if _, ok := manager.(transactionScopedManager); !ok {
			t.Fatalf("transaction manager type = %T, want transaction-scoped manager", manager)
		}
		if IsTxManager(manager) {
			t.Fatalf("transaction-scoped manager must not expose transaction lifecycle")
		}
		if _, ok := manager.(interface{ Commit() error }); ok {
			t.Fatalf("transaction-scoped manager unexpectedly exposes Commit")
		}
		if _, ok := manager.(interface{ Rollback() error }); ok {
			t.Fatalf("transaction-scoped manager unexpectedly exposes Rollback")
		}
		return nil
	}, tx.WithReadOnly(true))
	if err != nil {
		t.Fatalf("Transaction() error = %v", err)
	}
	if !handlerCalled {
		t.Fatal("transaction handler was not called")
	}
	if state.beginCalls != 1 || state.commitCalls != 1 || state.rollbackCalls != 0 {
		t.Fatalf("transaction calls = begin:%d commit:%d rollback:%d, want begin:1 commit:1 rollback:0", state.beginCalls, state.commitCalls, state.rollbackCalls)
	}
}

func TestTransactionPropagatesHandlerError_scope_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	engine := &Engine{db: db}
	handlerErr := errors.New("handler failed")

	err := Transaction(context.Background(), engine, func(context.Context, Manager) error {
		return handlerErr
	})
	if !errors.Is(err, handlerErr) {
		t.Fatalf("Transaction() error = %v, want %v", err, handlerErr)
	}
	if state.beginCalls != 1 || state.commitCalls != 0 || state.rollbackCalls != 1 {
		t.Fatalf("transaction calls = begin:%d commit:%d rollback:%d, want begin:1 commit:0 rollback:1", state.beginCalls, state.commitCalls, state.rollbackCalls)
	}
}

func TestTransactionCommitsOnSpecificError_scope_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	engine := &Engine{db: db}

	err := Transaction(context.Background(), engine, func(context.Context, Manager) error {
		return tx.ErrCommitOnSpecific
	})
	if !errors.Is(err, tx.ErrCommitOnSpecific) {
		t.Fatalf("Transaction() error = %v, want %v", err, tx.ErrCommitOnSpecific)
	}
	if state.beginCalls != 1 || state.commitCalls != 1 || state.rollbackCalls != 0 {
		t.Fatalf("transaction calls = begin:%d commit:%d rollback:%d, want begin:1 commit:1 rollback:0", state.beginCalls, state.commitCalls, state.rollbackCalls)
	}
}

func TestTransactionReusesScopedManager_scope_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	engine := &Engine{db: db}

	reusedHandlerCalled := false
	err := Transaction(context.Background(), engine, func(ctx context.Context, outer Manager) error {
		return Transaction(ctx, outer, func(_ context.Context, inner Manager) error {
			reusedHandlerCalled = true
			if inner != outer {
				t.Fatalf("inner manager = %p, want outer manager %p", inner, outer)
			}
			return nil
		}, tx.WithReadOnly(true))
	})
	if err != nil {
		t.Fatalf("Transaction() error = %v", err)
	}
	if !reusedHandlerCalled {
		t.Fatal("reused transaction handler was not called")
	}
	if state.beginCalls != 1 || state.commitCalls != 1 || state.rollbackCalls != 0 {
		t.Fatalf("transaction calls = begin:%d commit:%d rollback:%d, want begin:1 commit:1 rollback:0", state.beginCalls, state.commitCalls, state.rollbackCalls)
	}
}
