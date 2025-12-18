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

// Package juice defines the handler of the query and exec.

package juice

import (
	"context"

	"github.com/go-juicedev/juice/session"
	"github.com/go-juicedev/juice/sql"
)

// Handler defines a generic query handler function that executes database operations.
// It is a generic type that can handle different types of query results.
//
// Type Parameters:
//   - T: The return type of the handler function. Can be any type that represents
//     the result of a database operation (e.g., *sql.Rows, sql.Result).
//
// Parameters:
//   - ctx: Context for handling timeouts, cancellation, and passing values.
//   - query: The SQL query string to be executed.
//   - args: Variable number of arguments to be used in the query for parameter binding.
//
// Returns:
//   - T: The result of the query execution, type depends on the generic parameter T.
//   - error: Any error that occurred during query execution.
type Handler[T any] func(ctx context.Context, query string, args ...any) (T, error)

// QueryHandler is a specialized Handler type for query operations that return rows.
// It is specifically typed to return *sql.Rows, making it suitable for SELECT queries
// or any operation that returns a result set.
type QueryHandler = Handler[sql.Rows]

// ExecHandler is a specialized Handler type for execution operations.
// It is specifically typed to return sql.Result, making it suitable for
// INSERT, UPDATE, DELETE, or any other operation that modifies data.
type ExecHandler = Handler[sql.Result]

// GenericQueryHandler is a flexible query handler that can return custom result types.
// It allows for implementing custom result processing logic by specifying the desired
// return type through the generic parameter T.
//
// Type Parameters:
//   - T: The custom return type that the handler will produce.
type GenericQueryHandler[T any] Handler[T]

// SessionQueryHandler is the default QueryHandler.
// It will get the session from the context.
// And use the session to query the database.
func SessionQueryHandler(ctx context.Context, query string, args ...any) (sql.Rows, error) {
	sess, err := session.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return sess.QueryContext(ctx, query, args...)
}

// ensure SessionQueryHandler implements QueryHandler
var _ QueryHandler = SessionQueryHandler

// SessionExecHandler is the default ExecHandler.
// It will get the session from the context.
// And use the session to exec the database.
func SessionExecHandler(ctx context.Context, query string, args ...any) (sql.Result, error) {
	sess, err := session.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return sess.ExecContext(ctx, query, args...)
}

// ensure SessionExecHandler implements ExecHandler
var _ ExecHandler = SessionExecHandler
