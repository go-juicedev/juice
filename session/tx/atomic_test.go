package tx

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
)

type txDriverState struct {
	beginErr    error
	commitErr   error
	rollbackErr error

	beginCalled    int
	commitCalled   int
	rollbackCalled int
	lastBeginOpts  driver.TxOptions
}

type txDriverStub struct {
	state *txDriverState
}

func (d *txDriverStub) Open(_ string) (driver.Conn, error) {
	return &txConnStub{state: d.state}, nil
}

type txConnStub struct {
	state *txDriverState
}

func (c *txConnStub) Prepare(_ string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *txConnStub) Close() error {
	return nil
}

func (c *txConnStub) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *txConnStub) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.state.beginCalled++
	c.state.lastBeginOpts = opts
	if c.state.beginErr != nil {
		return nil, c.state.beginErr
	}
	return &txStub{state: c.state}, nil
}

var _ driver.ConnBeginTx = (*txConnStub)(nil)

type txStub struct {
	state *txDriverState
}

func (t *txStub) Commit() error {
	t.state.commitCalled++
	return t.state.commitErr
}

func (t *txStub) Rollback() error {
	t.state.rollbackCalled++
	return t.state.rollbackErr
}

var txDriverSeq uint64

func openTxTestDB(t *testing.T, state *txDriverState) *sql.DB {
	t.Helper()

	name := fmt.Sprintf("juice_tx_test_%d", atomic.AddUint64(&txDriverSeq, 1))
	sql.Register(name, &txDriverStub{state: state})

	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestTransactionOptionFuncs_atomic_test(t *testing.T) {
	options := &sql.TxOptions{}

	WithIsolationLevel(sql.LevelSerializable)(options)
	WithReadOnly(true)(options)

	if options.Isolation != sql.LevelSerializable {
		t.Fatalf("expected isolation level serializable, got %v", options.Isolation)
	}

	if !options.ReadOnly {
		t.Fatalf("expected readonly true")
	}
}

func TestAtomicContext_SuccessWithOptions_atomic_test(t *testing.T) {
	state := &txDriverState{}
	db := openTxTestDB(t, state)

	handlerCalled := false
	err := AtomicContext(context.Background(), db, func(_ context.Context, tx *sql.Tx) error {
		handlerCalled = true
		if tx == nil {
			t.Fatalf("expected non-nil tx")
		}
		return nil
	}, WithIsolationLevel(sql.LevelRepeatableRead), WithReadOnly(true))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Fatalf("expected handler to be called")
	}

	if state.beginCalled != 1 {
		t.Fatalf("expected begin called once, got %d", state.beginCalled)
	}

	if state.commitCalled != 1 {
		t.Fatalf("expected commit called once, got %d", state.commitCalled)
	}

	if state.lastBeginOpts.Isolation != driver.IsolationLevel(sql.LevelRepeatableRead) {
		t.Fatalf("unexpected isolation level: %v", state.lastBeginOpts.Isolation)
	}

	if !state.lastBeginOpts.ReadOnly {
		t.Fatalf("expected readonly option true")
	}
}

func TestAtomicContext_BeginError_atomic_test(t *testing.T) {
	beginErr := errors.New("begin failed")
	state := &txDriverState{beginErr: beginErr}
	db := openTxTestDB(t, state)

	handlerCalled := false
	err := AtomicContext(context.Background(), db, func(_ context.Context, _ *sql.Tx) error {
		handlerCalled = true
		return nil
	})

	if !errors.Is(err, beginErr) {
		t.Fatalf("expected begin error, got %v", err)
	}

	if handlerCalled {
		t.Fatalf("handler should not be called when begin fails")
	}

	if state.commitCalled != 0 || state.rollbackCalled != 0 {
		t.Fatalf("commit/rollback should not be called, got commit=%d rollback=%d", state.commitCalled, state.rollbackCalled)
	}
}

func TestAtomicContext_HandlerAndRollbackError_atomic_test(t *testing.T) {
	handlerErr := errors.New("handler failed")
	rollbackErr := errors.New("rollback failed")
	state := &txDriverState{rollbackErr: rollbackErr}
	db := openTxTestDB(t, state)

	err := AtomicContext(context.Background(), db, func(_ context.Context, _ *sql.Tx) error {
		return handlerErr
	})

	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected joined error contains handler error, got %v", err)
	}

	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected joined error contains rollback error, got %v", err)
	}

	if state.commitCalled != 0 {
		t.Fatalf("commit should not be called, got %d", state.commitCalled)
	}

	if state.rollbackCalled != 1 {
		t.Fatalf("rollback should be called once, got %d", state.rollbackCalled)
	}
}

func TestAtomicContext_CommitSpecificAndCommitError_atomic_test(t *testing.T) {
	commitErr := errors.New("commit failed")
	state := &txDriverState{commitErr: commitErr}
	db := openTxTestDB(t, state)

	err := AtomicContext(context.Background(), db, func(_ context.Context, _ *sql.Tx) error {
		return ErrCommitOnSpecific
	})

	if !errors.Is(err, ErrCommitOnSpecific) {
		t.Fatalf("expected joined error contains ErrCommitOnSpecific, got %v", err)
	}

	if !errors.Is(err, commitErr) {
		t.Fatalf("expected joined error contains commit error, got %v", err)
	}

	if state.commitCalled != 1 {
		t.Fatalf("commit should be called once, got %d", state.commitCalled)
	}
}

func TestAtomic_UsesBackgroundContext_atomic_test(t *testing.T) {
	state := &txDriverState{}
	db := openTxTestDB(t, state)

	err := Atomic(db, func(ctx context.Context, _ *sql.Tx) error {
		if ctx == nil {
			t.Fatalf("expected non-nil context")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
