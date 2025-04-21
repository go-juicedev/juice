/*
Copyright 2025 eatmoreapple

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

package tx

import (
	"context"
	"database/sql"
	"errors"
)

// ErrCommitOnSpecific is the error that commit on specific transaction.
var ErrCommitOnSpecific = errors.New("tx: commit on specific transaction")

// HandlerFunc is a function to execute a handler function within a database transaction.
type HandlerFunc func(ctx context.Context, tx *sql.Tx) error

// Atomic executes the given handler function within a database transaction.
func Atomic(ctx context.Context, db *sql.DB, h HandlerFunc, opts ...TransactionOptionFunc) (err error) {
	var (
		opt *sql.TxOptions
		tx  *sql.Tx
	)

	if len(opts) > 0 {
		opt = new(sql.TxOptions)
		for _, o := range opts {
			o(opt)
		}
	}

	tx, err = db.BeginTx(ctx, opt)
	if err != nil {
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

	if err = h(ctx, tx); err != nil {
		if !errors.Is(err, ErrCommitOnSpecific) {
			return err
		}
	}

	return errors.Join(err, tx.Commit())
}
