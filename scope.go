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

	"github.com/go-juicedev/juice/session/tx"
)

// ErrInvalidManager is an error for invalid manager.
var ErrInvalidManager = errors.New("juice: invalid manager")

// ErrCommitOnSpecific is an error for commit on specific transaction.
// Deprecated: use tx.ErrCommitOnSpecific instead.
var ErrCommitOnSpecific = tx.ErrCommitOnSpecific

// Transaction executes a transaction with the given handler.
// If the context does not carry an Engine, it will return ErrInvalidManager.
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
func Transaction(ctx context.Context, handler func(ctx context.Context) error, opts ...tx.TransactionOptionFunc) (err error) {
	engine, ok := engineFromContext(ctx)
	if !ok {
		return ErrInvalidManager
	}

	handlerFunc := tx.HandlerFunc(func(ctx context.Context, tx *sql.Tx) error {
		txManager := &BasicTxManager{
			basicTxManager: &basicTxManager{
				engine:      engine,
				ctx:         ctx,
				Transaction: tx,
			},
		}
		ctx = ContextWithManager(ctx, txManager)
		return handler(ctx)
	})

	return tx.AtomicContext(ctx, engine.DB(), handlerFunc, opts...)
}

// NestedTransaction executes the handler within the current transaction when one
// is already bound to the context.
//
// If the manager in ctx is a TxManager, the handler is executed directly and
// the existing transaction is reused. In this case, opts are ignored because
// the current transaction has already been started.
//
// If ctx is not in a transaction, NestedTransaction behaves like Transaction
// and starts a new transaction with opts applied.
func NestedTransaction(ctx context.Context, handler func(ctx context.Context) error, opts ...tx.TransactionOptionFunc) (err error) {
	manager, err := ManagerFromContext(ctx)
	if err != nil {
		return err
	}
	if IsTxManager(manager) {
		return handler(ctx)
	}
	return Transaction(ctx, handler, opts...)
}
