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

// TransactionHandler handles work using the provided transaction-scoped manager.
type TransactionHandler func(ctx context.Context, manager Manager) error

// Transaction executes the handler within a transaction.
//
// If manager is an Engine, Transaction starts a new transaction and applies
// opts. If manager is already transaction-scoped, Transaction reuses the
// existing transaction and ignores opts because an active transaction cannot
// be reconfigured. Other manager implementations return ErrInvalidManager.
//
// If the handler returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
// For example:
//
//	var engine *juice.Engine
//	// ... initialize engine
//	if err := juice.Transaction(ctx, engine, func(ctx context.Context, txManager juice.Manager) error {
//		_, err := juice.ExecContext(ctx, txManager, "User.Create", user)
//		return err
//	}); err != nil {
//		// handle error
//	}
func Transaction(ctx context.Context, manager Manager, handler TransactionHandler, opts ...tx.TransactionOptionFunc) (err error) {
	if manager == nil {
		return ErrInvalidManager
	}

	if _, ok := manager.(transactionScopedManager); ok {
		return handler(ctx, manager)
	}

	engine, ok := manager.(*Engine)
	if !ok || engine == nil {
		return ErrInvalidManager
	}

	handlerFunc := tx.HandlerFunc(func(ctx context.Context, transaction *sql.Tx) error {
		manager := &scopedManager{
			engine:  engine,
			session: transaction,
		}
		return handler(ctx, manager)
	})

	return tx.AtomicContext(ctx, engine.DB(), handlerFunc, opts...)
}
