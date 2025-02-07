/*
Copyright 2024 eatmoreapple

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
	"errors"
)

// ErrInvalidManager is an error for invalid manager.
var ErrInvalidManager = errors.New("juice: invalid manager")

// ErrCommitOnSpecific is an error for commit on specific transaction.
var ErrCommitOnSpecific = errors.New("juice: commit on specific transaction")

// TransactionOptionFunc is a function to set the transaction options.
// It is used to set the transaction options for the transaction.
type TransactionOptionFunc func(options *sql.TxOptions)

// WithIsolationLevel sets the isolation level for the transaction.
func WithIsolationLevel(level sql.IsolationLevel) TransactionOptionFunc {
	return func(options *sql.TxOptions) {
		options.Isolation = level
	}
}

// WithReadOnly sets the read-only flag for the transaction.
func WithReadOnly(readOnly bool) TransactionOptionFunc {
	return func(options *sql.TxOptions) {
		options.ReadOnly = readOnly
	}
}

// Transaction executes a transaction with the given handler.
// If the manager is not an instance of Engine, it will return ErrInvalidManager.
// If the handler returns an error, the transaction will be rolled back.
// Otherwise, the transaction will be committed.
// The ctx must should be created by ContextWithManager.
// For example:
//
//		var engine *juice.Engine
//		// ... initialize engine
//		ctx := juice.ContextWithManager(context.Background(), engine)
//	    if err := juice.Transaction(ctx, func(ctx context.Context) error {
//			// ... do something
//			return nil
//		}); err != nil {
//			// handle error
//		}
func Transaction(ctx context.Context, handler func(ctx context.Context) error, opts ...TransactionOptionFunc) (err error) {
	manager := ManagerFromContext(ctx)
	engine, ok := manager.(*Engine)
	if !ok {
		return ErrInvalidManager
	}

	var options *sql.TxOptions
	if len(opts) > 0 {
		options = new(sql.TxOptions)
		for _, opt := range opts {
			opt(options)
		}
	}
	// create a new transaction
	tx := engine.ContextTx(ctx, options)

	if err = tx.Begin(); err != nil {
		return err
	}
	defer func() {
		// make sure to roll back the transaction if there is an error
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			// if the error is not sql.ErrTxDone, it means the transaction is not already rolled back
			if !errors.Is(rollbackErr, sql.ErrTxDone) {
				err = errors.Join(err, rollbackErr)
			}
		}
	}()

	// create a new context with the transaction
	txCtx := ContextWithManager(ctx, tx)

	// call the handler
	err = handler(txCtx)
	if err != nil {
		// if the error is ErrCommitOnSpecific, it means the transaction needs to be committed by the user
		if !errors.Is(err, ErrCommitOnSpecific) {
			return err
		}
	}
	return errors.Join(err, tx.Commit())
}

// NestedTransaction executes a handler function with transaction support.
// If the manager is a TxManager, it will execute the handler within the existing transaction.
// Otherwise, it will create a new transaction and execute the handler within the new transaction.
func NestedTransaction(ctx context.Context, handler func(ctx context.Context) error, opts ...TransactionOptionFunc) (err error) {
	manager := ManagerFromContext(ctx)
	if IsTxManager(manager) {
		return handler(ctx)
	}
	return Transaction(ctx, handler, opts...)
}
