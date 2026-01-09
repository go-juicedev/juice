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

package juice

import (
	"context"
	"database/sql"
	sqllib "github.com/go-juicedev/juice/sql"
)

// This file provides context-based database helper shortcuts.

// QueryContext executes a query with the provided context and scans a single result into T.
// (ctx must contain a Manager via ManagerFromContext)
func QueryContext[T any](ctx context.Context, statement, param any) (result T, err error) {
	manager, err := ManagerFromContext(ctx)
	if err != nil {
		return result, err
	}
	executor := NewGenericManager[T](manager).Object(statement)
	return executor.QueryContext(ctx, param)
}

// ExecContext executes a statement that does not return rows and returns a sql.Result.
// (ctx must contain a Manager via ManagerFromContext)
func ExecContext(ctx context.Context, statement, param any) (result sql.Result, err error) {
	manager, err := ManagerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	executor := manager.Object(statement)
	return executor.ExecContext(ctx, param)
}

// QueryListContext executes a query and returns a slice of T. Rows are closed after reading.
// (ctx must contain a Manager via ManagerFromContext)
func QueryListContext[T any](ctx context.Context, statement, param any) (result []T, err error) {
	manager, err := ManagerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := manager.Object(statement).QueryContext(ctx, param)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return sqllib.List[T](rows)
}

// QueryList2Context executes a query and returns a slice of pointers to T. Rows are closed after reading.
// (ctx must contain a Manager via ManagerFromContext)
func QueryList2Context[T any](ctx context.Context, statement, param any) (result []*T, err error) {
	manager, err := ManagerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := manager.Object(statement).QueryContext(ctx, param)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return sqllib.List2[T](rows)
}

// QueryIterContext executes a query and returns an iterator over T.
// Rows are automatically closed when iteration completes or stops.
// (ctx must contain a Manager via ManagerFromContext)
//
// IMPORTANT: The returned iterator MUST be iterated over (even partially),
// otherwise the underlying database rows will not be closed, leading to resource leaks.
func QueryIterContext[T any](ctx context.Context, statement, param any) (sqllib.Iterator[T], error) {
	manager, err := ManagerFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := manager.Object(statement).QueryContext(ctx, param)
	if err != nil {
		return nil, err
	}

	iterator, err := sqllib.Iter[T](rows)
	if err != nil {
		_ = rows.Close()
		return nil, err
	}

	// Wrap the iterator to ensure rows are closed after iteration
	return func(yield func(T, error) bool) {
		defer func() { _ = rows.Close() }()
		iterator(yield)
	}, nil
}
