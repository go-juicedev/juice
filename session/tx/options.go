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

import "database/sql"

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
