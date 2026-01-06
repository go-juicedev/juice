/*
Copyright 2023 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package juice

import (
	"context"
	"database/sql"

	"github.com/go-juicedev/juice/session"
	"github.com/go-juicedev/juice/session/tx"
)

// Manager is an interface for managing database operations.
// It provides a high-level abstraction for executing SQL operations
// through the Object method which returns a SQLRowsExecutor.
type Manager interface {
	Object(v any) SQLRowsExecutor
}

// NewGenericManager returns a new GenericManager.
func NewGenericManager[T any](manager Manager) *GenericManager[T] {
	return &GenericManager[T]{Manager: manager}
}

// GenericManager is a generic manager for a specific type T
// that provides type-safe database operations.
type GenericManager[T any] struct {
	Manager
}

// Object implements the GenericManager interface.
func (s *GenericManager[T]) Object(v any) Executor[T] {
	exe := &GenericExecutor[T]{SQLRowsExecutor: s.Manager.Object(v)}
	return exe
}

// TxManager is a transactional manager that extends the base Manager interface
// with transaction control capabilities. It provides methods for beginning,
// committing, and rolling back database transactions.
type TxManager interface {
	Manager

	// Begin begins a new database transaction.
	// Returns an error if transaction is already started or if there's a database error.
	Begin() error

	// Commit commits the current transaction.
	// Returns an error if there's no active transaction or if commit fails.
	Commit() error

	// Rollback aborts the current transaction.
	// Returns an error if there's no active transaction or if rollback fails.
	Rollback() error
}

type basicTxManager struct {
	// Transaction holds the current transaction session
	// It's nil if no transaction is active
	session.Transaction

	ctx context.Context
	// engine is the database engine instance that handles database operations
	engine *Engine
}

func (b *basicTxManager) Object(v any) SQLRowsExecutor {
	statement, err := b.engine.GetConfiguration().GetStatement(v)
	if err != nil {
		return inValidExecutor(err)
	}
	drv := b.engine.Driver()
	statementHandler := newBatchStatementHandler(drv, b.Transaction, b.engine.GetConfiguration(), b.engine.middlewares...)
	return NewSQLRowsExecutor(statement, statementHandler, drv)
}

// BasicTxManager implements the TxManager interface providing basic
// transaction management functionality.
type BasicTxManager struct {
	*basicTxManager

	// txOptions configures the transaction behavior
	// If nil, default database transaction options are used
	txOptions *sql.TxOptions
}

// Object implements the Manager interface
func (t *BasicTxManager) Object(v any) SQLRowsExecutor {
	if t.Transaction == nil {
		return inValidExecutor(tx.ErrTransactionNotBegun)
	}
	return t.basicTxManager.Object(v)
}

// Begin begins the transaction
func (t *BasicTxManager) Begin() (err error) {
	// If the transaction is already begun, return an error directly.
	if t.Transaction != nil {
		return tx.ErrTransactionAlreadyBegun
	}
	t.Transaction, err = t.engine.DB().BeginTx(t.ctx, t.txOptions)
	return err
}

// Commit commits the transaction
func (t *BasicTxManager) Commit() error {
	// If the transaction is not begun, return an error directly.
	if t.Transaction == nil {
		return tx.ErrTransactionNotBegun
	}
	return t.Transaction.Commit()
}

// Rollback rollbacks the transaction
func (t *BasicTxManager) Rollback() error {
	// If the transaction is not begun, return an error directly.
	if t.Transaction == nil {
		return tx.ErrTransactionNotBegun
	}
	return t.Transaction.Rollback()
}

func (t *BasicTxManager) Raw(query string) Runner {
	if t.Transaction == nil {
		return NewErrorRunner(tx.ErrTransactionNotBegun)
	}
	return NewRunner(query, t.engine, t.Transaction)
}

type managerKey struct{}

// managerFromContext returns the Manager from the context.
func managerFromContext(ctx context.Context) (Manager, bool) {
	manager, ok := ctx.Value(managerKey{}).(Manager)
	return manager, ok
}

// ManagerFromContext returns the Manager from the context.
func ManagerFromContext(ctx context.Context) (Manager, error) {
	manager, ok := managerFromContext(ctx)
	if !ok {
		return nil, ErrNoManagerFoundInContext
	}
	return manager, nil
}

// ContextWithManager returns a new context with the given Manager.
func ContextWithManager(ctx context.Context, manager Manager) context.Context {
	return context.WithValue(ctx, managerKey{}, manager)
}

// IsTxManager returns true if the manager is a TxManager.
func IsTxManager(manager Manager) bool {
	_, ok := manager.(TxManager)
	return ok
}
