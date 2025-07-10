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
	"fmt"
	"reflect"
	"strconv"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/internal/ctxreducer"
	"github.com/go-juicedev/juice/internal/reflectlite"
	"github.com/go-juicedev/juice/internal/stmt"
	"github.com/go-juicedev/juice/session"
)

// StatementHandler is an interface that defines methods for executing SQL statements.
// It provides two methods: ExecContext and QueryContext, which are used to execute
// non-query and query SQL statements respectively.
type StatementHandler interface {
	// ExecContext executes a non-query SQL statement (such as INSERT, UPDATE, DELETE)
	// within a context, and returns the result. It takes a context, a Statement object,
	// and a Param object as parameters.
	ExecContext(ctx context.Context, statement Statement, param Param) (sql.Result, error)

	// QueryContext executes a query SQL statement (such as SELECT) within a context,
	// and returns the resulting rows. It takes a context, a Statement object, and a
	// Param object as parameters.
	QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error)
}

// CompiledStatementHandler handles pre-built SQL statements execution.
// It maintains the pre-built query and arguments to avoid rebuilding them
// for each execution, improving performance for frequently used queries.
type CompiledStatementHandler struct {
	query        string
	args         []any
	middlewares  MiddlewareGroup
	driver       driver.Driver
	session      session.Session
	queryHandler QueryHandler
	execHandler  ExecHandler
}

// QueryContext executes a query that returns rows. It enriches the context with
// session and parameter information, then executes the pre-built query through
// the middleware chain using SessionQueryHandler.
func (s *CompiledStatementHandler) QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error) {
	contextReducer := ctxreducer.G{
		ctxreducer.NewSessionContextReducer(s.session),
		ctxreducer.NewParamContextReducer(param),
	}
	ctx = contextReducer.Reduce(ctx)
	if s.queryHandler == nil {
		s.queryHandler = SessionQueryHandler
	}
	return s.middlewares.QueryContext(statement, s.queryHandler)(ctx, s.query, s.args...)
}

// ExecContext executes a non-query SQL statement (such as INSERT, UPDATE, DELETE)
// within a context. Similar to QueryContext, it enriches the context and executes
// the pre-built query through the middleware chain using SessionExecHandler.
func (s *CompiledStatementHandler) ExecContext(ctx context.Context, statement Statement, param Param) (sql.Result, error) {
	contextReducer := ctxreducer.G{
		ctxreducer.NewSessionContextReducer(s.session),
		ctxreducer.NewParamContextReducer(param),
	}
	ctx = contextReducer.Reduce(ctx)
	if s.execHandler == nil {
		s.execHandler = SessionExecHandler
	}
	return s.middlewares.ExecContext(statement, s.execHandler)(ctx, s.query, s.args...)
}

// PreparedStatementHandler implements the StatementHandler interface.
// It maintains a single prepared statement that can be reused if the query is the same.
// When a different query is encountered, it closes the existing statement and creates a new one.
type PreparedStatementHandler struct {
	stmts       *sql.Stmt
	middlewares MiddlewareGroup
	driver      driver.Driver
	session     session.Session
}

// getOrPrepare retrieves an existing prepared statement if the query matches,
// otherwise closes the current statement (if any) and creates a new one.
func (s *PreparedStatementHandler) getOrPrepare(ctx context.Context, query string) (*sql.Stmt, error) {
	if s.stmts != nil && stmt.Query(s.stmts) == query {
		return s.stmts, nil
	}
	// it means the prepared statement is not what we want
	if s.stmts != nil {
		_ = s.stmts.Close()
	}
	var err error
	s.stmts, err = s.session.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare statement failed: %w", err)
	}
	return s.stmts, nil
}

// QueryContext executes a query that returns rows. It builds the query using
// the provided Statement and Param, applies middlewares, and executes the
// prepared statement with the given context.
func (s *PreparedStatementHandler) QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error) {
	query, args, err := statement.Build(s.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	queryHandler := func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		preparedStmt, err := s.getOrPrepare(ctx, query)
		if err != nil {
			return nil, err
		}
		return preparedStmt.QueryContext(ctx, args...)
	}
	statementHandler := CompiledStatementHandler{
		query:        query,
		args:         args,
		middlewares:  s.middlewares,
		driver:       s.driver,
		session:      s.session,
		queryHandler: queryHandler,
	}
	return statementHandler.QueryContext(ctx, statement, param)
}

// ExecContext executes a query that doesn't return rows. It builds the query
// using the provided Statement and Param, applies middlewares, and executes
// the prepared statement with the given context.
func (s *PreparedStatementHandler) ExecContext(ctx context.Context, statement Statement, param Param) (result sql.Result, err error) {
	query, args, err := statement.Build(s.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	execHandler := func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		preparedStmt, err := s.getOrPrepare(ctx, query)
		if err != nil {
			return nil, err
		}
		return preparedStmt.ExecContext(ctx, args...)
	}
	statementHandler := CompiledStatementHandler{
		query:       query,
		args:        args,
		middlewares: s.middlewares,
		driver:      s.driver,
		session:     s.session,
		execHandler: execHandler,
	}
	return statementHandler.ExecContext(ctx, statement, param)
}

// Close closes all prepared statements in the pool and returns any error
// that occurred during the process. Multiple errors are joined together.
func (s *PreparedStatementHandler) Close() error {
	if s.stmts != nil {
		return s.stmts.Close()
	}
	return nil
}

// QueryBuildStatementHandler handles the execution of SQL statements and returns
// the results in a sql.Rows structure. It integrates a driver, middlewares, and
// a session to manage the execution flow.
type QueryBuildStatementHandler struct {
	driver      driver.Driver
	middlewares MiddlewareGroup
	session     session.Session
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (s *QueryBuildStatementHandler) QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error) {
	query, args, err := statement.Build(s.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	statementHandler := CompiledStatementHandler{
		query:       query,
		args:        args,
		middlewares: s.middlewares,
		driver:      s.driver,
		session:     s.session,
	}
	return statementHandler.QueryContext(ctx, statement, param)
}

// ExecContext executes a non-query SQL statement (such as INSERT, UPDATE, DELETE)
// within a context, and returns the result. Similar to QueryContext, it constructs
// the SQL command, applies middlewares, and executes the command using the driver.
func (s *QueryBuildStatementHandler) ExecContext(ctx context.Context, statement Statement, param Param) (sql.Result, error) {
	query, args, err := statement.Build(s.driver.Translator(), param)
	if err != nil {
		return nil, err
	}
	statementHandler := CompiledStatementHandler{
		query:       query,
		args:        args,
		middlewares: s.middlewares,
		driver:      s.driver,
		session:     s.session,
	}
	return statementHandler.ExecContext(ctx, statement, param)
}

var _ StatementHandler = (*QueryBuildStatementHandler)(nil)

// NewQueryBuildStatementHandler creates a new instance of QueryBuildStatementHandler
// with the provided driver, session, and an optional list of middlewares. This
// function is typically used to initialize the handler before executing SQL statements.
func NewQueryBuildStatementHandler(driver driver.Driver, session session.Session, middlewares ...Middleware) StatementHandler {
	return &QueryBuildStatementHandler{
		driver:      driver,
		middlewares: middlewares,
		session:     session,
	}
}

var errInvalidParamType = errors.New("invalid param type")

// ErrBatchSkip is a sentinel error that indicates batch processing should skip
// the current error and continue executing subsequent batches. When this error
// is returned (or wrapped) from middleware or statement execution, the batch
// handler will collect the error but continue processing remaining batches
// instead of immediately returning.
//
// Usage:
//   - Return directly: return ErrBatchSkip
//   - Wrap with context: return fmt.Errorf("%w: connection timeout", ErrBatchSkip)
//   - Check with errors.Is(): if errors.Is(err, ErrBatchSkip) { /* handle gracefully */ }
//
// The batch handler uses errors.Is() to detect this error and will:
//  1. Collect the error using errors.Join()
//  2. Continue to the next batch instead of stopping
//  3. Return all collected errors at the end of batch processing
//
// This allows for resilient batch operations where individual batch failures
// don't prevent the entire operation from completing. Middleware can use this
// error to implement custom retry logic, connection failover, or other
// error recovery strategies during batch processing.
var ErrBatchSkip = errors.New("skip batch error and continue")

type sliceBatchStatementHandler struct {
	driver      driver.Driver
	middlewares MiddlewareGroup
	session     session.Session
	value       reflect.Value
	batchSize   int64
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (s *sliceBatchStatementHandler) QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error) {
	statementHandler := NewQueryBuildStatementHandler(s.driver, s.session, s.middlewares...)
	return statementHandler.QueryContext(ctx, statement, param)
}

func (s *sliceBatchStatementHandler) execContext(ctx context.Context, statement Statement, param Param) (sql.Result, error) {
	statementHandler := NewQueryBuildStatementHandler(s.driver, s.session, s.middlewares...)
	return statementHandler.ExecContext(ctx, statement, param)
}

func (s *sliceBatchStatementHandler) ExecContext(ctx context.Context, statement Statement, param Param) (result sql.Result, err error) {
	length := s.value.Len()
	if length == 0 {
		return nil, fmt.Errorf("%w: empty slice", errInvalidParamType)
	}
	times := (length + int(s.batchSize) - 1) / int(s.batchSize)

	if times == 1 {
		return s.execContext(ctx, statement, param)
	}

	// Create a PreparedStatementHandler for batch processing.
	// We use PreparedStatementHandler here because:
	// 1. For batch inserts with size N, we only need at most 2 prepared statements:
	//    - One for full batch (N rows)
	//    - One for remaining rows (< N rows)
	// 2. These statements can be reused across multiple batches
	// 3. This significantly reduces the overhead of preparing statements repeatedly
	preparedStatementHandler := &PreparedStatementHandler{
		driver:      s.driver,
		middlewares: s.middlewares,
		session:     s.session,
	}

	// Ensure all prepared statements are properly closed after use
	defer func() { _ = preparedStatementHandler.Close() }()

	var batchErrs error
	// execute the statement in batches.
	for i := 0; i < times; i++ {
		start := i * int(s.batchSize)
		end := (i + 1) * int(s.batchSize)
		if end > length {
			end = length
		}
		batchParam := s.value.Slice(start, end).Interface()
		result, err = preparedStatementHandler.ExecContext(ctx, statement, batchParam)
		if err != nil {
			if errors.Is(err, ErrBatchSkip) {
				batchErrs = errors.Join(batchErrs, err)
				continue
			}
			return nil, err
		}
	}

	if batchErrs != nil {
		return nil, batchErrs
	}
	return result, batchErrs
}

type mapBatchStatementHandler struct {
	driver      driver.Driver
	middlewares MiddlewareGroup
	session     session.Session
	value       reflect.Value
	batchSize   int64
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (s *mapBatchStatementHandler) QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error) {
	statementHandler := NewQueryBuildStatementHandler(s.driver, s.session, s.middlewares...)
	return statementHandler.QueryContext(ctx, statement, param)
}

func (s *mapBatchStatementHandler) execContext(ctx context.Context, statement Statement, param Param) (sql.Result, error) {
	statementHandler := NewQueryBuildStatementHandler(s.driver, s.session, s.middlewares...)
	return statementHandler.ExecContext(ctx, statement, param)
}

func (s *mapBatchStatementHandler) ExecContext(ctx context.Context, statement Statement, param Param) (result sql.Result, err error) {
	mapKeys := s.value.MapKeys()
	if len(mapKeys) != 1 {
		return nil, fmt.Errorf("%w: expected one key, got %d", errInvalidParamType, len(mapKeys))
	}
	keyValue := mapKeys[0]
	if keyValue.Kind() != reflect.String {
		return nil, fmt.Errorf("%w: expected string key, got %s", errInvalidParamType, keyValue.Kind())
	}
	value := s.value.MapIndex(keyValue)
	value = reflectlite.Unpack(value)
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return nil, fmt.Errorf("%w: map value must be slice or array, got %s", errInvalidParamType, value.Kind())
	}
	length := value.Len()
	if length == 0 {
		return nil, fmt.Errorf("%w: empty slice", errInvalidParamType)
	}
	times := (length + int(s.batchSize) - 1) / int(s.batchSize)

	if times == 1 {
		return s.execContext(ctx, statement, param)
	}

	// Create a PreparedStatementHandler for batch processing.
	// We use PreparedStatementHandler here because:
	// 1. For batch inserts with size N, we only need at most 2 prepared statements:
	//    - One for full batch (N rows)
	//    - One for remaining rows (< N rows)
	// 2. These statements can be reused across multiple batches
	// 3. This significantly reduces the overhead of preparing statements repeatedly
	preparedStatementHandler := &PreparedStatementHandler{
		driver:      s.driver,
		middlewares: s.middlewares,
		session:     s.session,
	}

	// Ensure all prepared statements are properly closed after use
	defer func() { _ = preparedStatementHandler.Close() }()

	var batchErrs error

	batchParam := reflect.MakeMap(s.value.Type())
	executionParam := batchParam.Interface()

	// execute the statement in batches.
	for i := 0; i < times; i++ {
		start := i * int(s.batchSize)
		end := (i + 1) * int(s.batchSize)
		if end > length {
			end = length
		}
		batchParam.SetMapIndex(keyValue, value.Slice(start, end))
		result, err = preparedStatementHandler.ExecContext(ctx, statement, executionParam)
		if err != nil {
			if errors.Is(err, ErrBatchSkip) {
				batchErrs = errors.Join(batchErrs, err)
				continue
			}
			return nil, err
		}
	}

	if batchErrs != nil {
		return nil, batchErrs
	}
	return result, nil
}

// BatchStatementHandler is a specialized SQL statement executor that provides optimized handling
// of batch operations, particularly for INSERT statements. It supports both single and batch
// execution modes, automatically switching to batch processing when:
// 1. The statement is an INSERT operation
// 2. A batch size is specified in the configuration
// 3. The input parameters represent multiple records (slice or map of structs)
//
// The handler integrates with the middleware chain and supports both regular and batch
// execution contexts. For non-batch operations, it behaves similarly to QueryBuildStatementHandler.
type BatchStatementHandler struct {
	driver      driver.Driver   // The driver used to execute SQL statements.
	middlewares MiddlewareGroup // The group of middlewares to apply to the SQL statements.
	session     session.Session // The session used to manage the database connection.
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (b *BatchStatementHandler) QueryContext(ctx context.Context, statement Statement, param Param) (*sql.Rows, error) {
	statementHandler := NewQueryBuildStatementHandler(b.driver, b.session, b.middlewares...)
	return statementHandler.QueryContext(ctx, statement, param)
}

// ExecContext executes a batch of SQL statements within a context. It handles
// the execution of SQL statements in batches if the action is an Insert and a
// batch size is specified. If the action is not an Insert or no batch size is
// specified, it delegates to the execContext method.
func (b *BatchStatementHandler) ExecContext(ctx context.Context, statement Statement, param Param) (result sql.Result, err error) {
	if statement.Action() != Insert {
		return b.execContext(ctx, statement, param)
	}
	batchSizeValue := statement.Attribute("batchSize")
	if len(batchSizeValue) == 0 {
		return b.execContext(ctx, statement, param)
	}
	batchSize, err := strconv.ParseInt(batchSizeValue, 10, 64)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("failed to parse batch size: %s", batchSizeValue))
	}
	if batchSize <= 0 {
		return nil, errors.New("batch size must be greater than 0")
	}

	var statementHandler StatementHandler

	// ensure the param is a slice or array
	value := reflectlite.ValueOf(param)

	switch value.IndirectType().Kind() {
	case reflect.Slice, reflect.Array:
		statementHandler = &sliceBatchStatementHandler{
			driver:      b.driver,
			middlewares: b.middlewares,
			session:     b.session,
			batchSize:   batchSize,
			value:       value.Unwrap().Value,
		}
	case reflect.Map:
		statementHandler = &mapBatchStatementHandler{
			driver:      b.driver,
			middlewares: b.middlewares,
			session:     b.session,
			batchSize:   batchSize,
			value:       value.Unwrap().Value,
		}
	default:
		return nil, errSliceOrArrayRequired
	}
	return statementHandler.ExecContext(ctx, statement, param)
}

func (b *BatchStatementHandler) execContext(ctx context.Context, statement Statement, param Param) (sql.Result, error) {
	statementHandler := NewQueryBuildStatementHandler(b.driver, b.session, b.middlewares...)
	return statementHandler.ExecContext(ctx, statement, param)
}

// NewBatchStatementHandler returns a new instance of StatementHandler with the default behavior.
func NewBatchStatementHandler(driver driver.Driver, session session.Session, middlewares ...Middleware) StatementHandler {
	return &BatchStatementHandler{
		driver:      driver,
		middlewares: middlewares,
		session:     session,
	}
}
