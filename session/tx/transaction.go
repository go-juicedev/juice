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
	"errors"
)

var (
	// ErrTransactionAlreadyBegun is the error that transaction already begun.
	ErrTransactionAlreadyBegun = errors.New("tx: transaction already begun")

	// ErrTransactionNotBegun is the error that transaction not begun.
	ErrTransactionNotBegun = errors.New("tx: transaction not begun")
)

// Transaction is a interface that can be used to commit and rollback.
type Transaction interface {
	Commit() error
	Rollback() error
}
