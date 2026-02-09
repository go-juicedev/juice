package juice

import (
	"context"
	"errors"
	"testing"

	"github.com/go-juicedev/juice/session/tx"
)

func TestScopeTransactionAndNestedTransaction_scope_test(t *testing.T) {
	if err := Transaction(context.Background(), func(context.Context) error { return nil }); !errors.Is(err, ErrNoManagerFoundInContext) {
		t.Fatalf("expected ErrNoManagerFoundInContext, got %v", err)
	}

	invalidCtx := ContextWithManager(context.Background(), &managerStub{})
	if err := Transaction(invalidCtx, func(context.Context) error { return nil }); !errors.Is(err, ErrInvalidManager) {
		t.Fatalf("expected ErrInvalidManager, got %v", err)
	}

	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	engine := &Engine{db: db}
	ctx := ContextWithManager(context.Background(), engine)

	handlerCalled := false
	err := Transaction(ctx, func(ctx context.Context) error {
		handlerCalled = true
		manager, err := ManagerFromContext(ctx)
		if err != nil {
			t.Fatalf("expected manager in transaction context: %v", err)
		}
		txManager, ok := manager.(*basicTxManager)
		if !ok {
			t.Fatalf("expected basicTxManager in transaction context, got %T", manager)
		}
		if txManager.Transaction == nil {
			t.Fatalf("expected active sql transaction in context manager")
		}
		return nil
	}, tx.WithReadOnly(true))
	if err != nil {
		t.Fatalf("unexpected transaction error: %v", err)
	}
	if !handlerCalled {
		t.Fatalf("expected transaction handler called")
	}

	handlerErr := errors.New("handler failed")
	err = Transaction(ctx, func(context.Context) error { return handlerErr })
	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected handler error, got %v", err)
	}

	err = Transaction(ctx, func(context.Context) error { return ErrCommitOnSpecific })
	if !errors.Is(err, ErrCommitOnSpecific) {
		t.Fatalf("expected ErrCommitOnSpecific, got %v", err)
	}

	nestedCalled := false
	txManagerCtx := ContextWithManager(context.Background(), &BasicTxManager{basicTxManager: &basicTxManager{}})
	err = NestedTransaction(txManagerCtx, func(context.Context) error {
		nestedCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected nested transaction error: %v", err)
	}
	if !nestedCalled {
		t.Fatalf("expected nested transaction handler called")
	}

	err = NestedTransaction(ctx, func(context.Context) error { return nil })
	if err != nil {
		t.Fatalf("unexpected nested transaction with engine error: %v", err)
	}
}

