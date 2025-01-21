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

	"github.com/go-juicedev/juice/session"
)

// Runner defines the interface for SQL operations.
// It provides methods for executing SELECT, INSERT, UPDATE, and DELETE operations.
// Implementations of this interface should handle SQL query execution and result processing.
type Runner interface {
	// Select executes a SELECT query and returns the result rows.
	Select(ctx context.Context, param Param) (*sql.Rows, error)

	// Insert executes an INSERT statement and returns the result.
	// The result includes the last insert ID and number of rows affected.
	Insert(ctx context.Context, param Param) (sql.Result, error)

	// Update executes an UPDATE statement and returns the result.
	// The result includes the number of rows affected by the update.
	Update(ctx context.Context, param Param) (sql.Result, error)

	// Delete executes a DELETE statement and returns the result.
	// The result includes the number of rows affected by the deletion.
	Delete(ctx context.Context, param Param) (sql.Result, error)
}

// ErrorRunner is a Runner implementation that always returns an error.
// It's useful for handling invalid states, configuration errors, or when operations
// should be prevented from executing. All methods return the same error that was
// provided during creation.
type ErrorRunner struct {
	err error
}

// Select implements Runner.Select by returning the stored error.
// It ignores the context and parameters, always returning nil for rows and the stored error.
func (r *ErrorRunner) Select(_ context.Context, _ Param) (*sql.Rows, error) {
	return nil, r.err
}

// Insert implements Runner.Insert by returning the stored error.
// It ignores the context and parameters, always returning nil for result and the stored error.
func (r *ErrorRunner) Insert(_ context.Context, _ Param) (sql.Result, error) {
	return nil, r.err
}

// Update implements Runner.Update by returning the stored error.
// It ignores the context and parameters, always returning nil for result and the stored error.
func (r *ErrorRunner) Update(_ context.Context, _ Param) (sql.Result, error) {
	return nil, r.err
}

// Delete implements Runner.Delete by returning the stored error.
// It ignores the context and parameters, always returning nil for result and the stored error.
func (r *ErrorRunner) Delete(_ context.Context, _ Param) (sql.Result, error) {
	return nil, r.err
}

// NewErrorRunner creates a new ErrorRunner that always returns the specified error.
// This is useful for creating a Runner that represents a failed state, such as
// when initialization fails or when operations should be prevented.
func NewErrorRunner(err error) Runner {
	return &ErrorRunner{err: err}
}

// SQLRunner is the standard implementation of Runner interface.
// It holds the SQL query, engine configuration, and session information.
type SQLRunner struct {
	query   string
	engine  *Engine
	session session.Session
}

// BuildExecutor creates a new SQL executor based on the given action.
// It configures the statement handler with the necessary driver and middleware.
func (r *SQLRunner) BuildExecutor(action Action) Executor[*sql.Rows] {
	statement := rawSQLStatement{
		query:  r.query,
		cfg:    r.engine.GetConfiguration(),
		action: action,
	}
	statementHandler := &QueryBuildStatementHandler{
		driver:      r.engine.driver,
		middlewares: r.engine.middlewares,
		session:     r.session,
	}
	return &sqlRowsExecutor{
		statement:        statement,
		statementHandler: statementHandler,
		driver:           r.engine.driver,
	}
}

// queryContext executes a SELECT query with the given context and parameters.
// It returns the query results as sql.Rows and any error that occurred.
func (r *SQLRunner) queryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	executor := r.BuildExecutor(Select)
	return executor.QueryContext(ctx, param)
}

// execContext executes a non-query SQL operation (INSERT, UPDATE, DELETE)
// with the given context and parameters.
func (r *SQLRunner) execContext(action Action, ctx context.Context, param Param) (sql.Result, error) {
	executor := r.BuildExecutor(action)
	return executor.ExecContext(ctx, param)
}

// Select executes a SELECT query and returns the result rows.
func (r *SQLRunner) Select(ctx context.Context, param Param) (*sql.Rows, error) {
	return r.queryContext(ctx, param)
}

// Insert executes an INSERT statement and returns the result.
func (r *SQLRunner) Insert(ctx context.Context, param Param) (sql.Result, error) {
	return r.execContext(Insert, ctx, param)
}

// Update executes an UPDATE statement and returns the result.
func (r *SQLRunner) Update(ctx context.Context, param Param) (sql.Result, error) {
	return r.execContext(Update, ctx, param)
}

// Delete executes a DELETE statement and returns the result.
func (r *SQLRunner) Delete(ctx context.Context, param Param) (sql.Result, error) {
	return r.execContext(Delete, ctx, param)
}

// NewRunner creates a new SQLRunner instance with the specified query, engine, and session.
func NewRunner(query string, engine *Engine, session session.Session) Runner {
	return &SQLRunner{
		query:   query,
		engine:  engine,
		session: session,
	}
}

// GenericRunner is a generic Runner implementation that binds the result of a SELECT query to a value of type T.
type GenericRunner[T any] struct {
	Runner
}

// Bind binds the result of a SELECT query to a single value of type T.
// It executes the query with the given context and parameters, then binds the result.
func (r *GenericRunner[T]) Bind(ctx context.Context, param Param) (result T, err error) {
	rows, err := r.Runner.Select(ctx, param)
	if err != nil {
		return result, err
	}
	defer func() { _ = rows.Close() }()
	return Bind[T](rows)
}

// List binds the result of a SELECT query to a list of values of type T.
// It executes the query with the given context and parameters, then binds the result.
func (r *GenericRunner[T]) List(ctx context.Context, param Param) (result []T, err error) {
	rows, err := r.Runner.Select(ctx, param)
	if err != nil {
		return result, err
	}
	defer func() { _ = rows.Close() }()
	return List[T](rows)
}

// List2 binds the result of a SELECT query to a list of pointers to values of type T.
// It executes the query with the given context and parameters, then binds the result.
func (r *GenericRunner[T]) List2(ctx context.Context, param Param) (result []*T, err error) {
	rows, err := r.Runner.Select(ctx, param)
	if err != nil {
		return result, err
	}
	defer func() { _ = rows.Close() }()
	return List2[T](rows)
}

// NewGenericRunner creates a new GenericRunner instance with the specified query, engine, and session.
func NewGenericRunner[T any](query string, engine *Engine, session session.Session) *GenericRunner[T] {
	return &GenericRunner[T]{
		Runner: NewRunner(query, engine, session),
	}
}
