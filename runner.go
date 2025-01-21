package juice

import (
	"context"
	"database/sql"

	"github.com/go-juicedev/juice/session"
)

// Runner defines the interface for SQL operations.
// It provides methods for executing SELECT, INSERT, UPDATE, and DELETE operations.
type Runner interface {
	Select(ctx context.Context, param Param) (*sql.Rows, error)
	Insert(ctx context.Context, param Param) (sql.Result, error)
	Update(ctx context.Context, param Param) (sql.Result, error)
	Delete(ctx context.Context, param Param) (sql.Result, error)
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

// ErrorRunner is a Runner implementation that always returns an error.
// Useful for handling invalid states or configurations.
type ErrorRunner struct {
	err error
}

// Select executes a SELECT query and returns the result rows.
// It always returns an error.
func (r *ErrorRunner) Select(_ context.Context, _ Param) (*sql.Rows, error) {
	return nil, r.err
}

// Insert executes an INSERT statement and returns the result.
// It always returns an error.
func (r *ErrorRunner) Insert(_ context.Context, _ Param) (sql.Result, error) {
	return nil, r.err
}

// Update executes an UPDATE statement and returns the result.
// It always returns an error.
func (r *ErrorRunner) Update(_ context.Context, _ Param) (sql.Result, error) {
	return nil, r.err
}

// Delete executes a DELETE statement and returns the result.
// It always returns an error.
func (r *ErrorRunner) Delete(_ context.Context, _ Param) (sql.Result, error) {
	return nil, r.err
}

// NewErrorRunner creates a new ErrorRunner that always returns the specified error.
func NewErrorRunner(err error) Runner {
	return &ErrorRunner{err: err}
}
