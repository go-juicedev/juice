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
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/internal/ctxreducer"
	"github.com/go-juicedev/juice/internal/reflectlite"
	"github.com/go-juicedev/juice/internal/stmt"
	"github.com/go-juicedev/juice/session"
	sqllib "github.com/go-juicedev/juice/sql"
)

// StatementHandler defines the interface for executing SQL statements.
type StatementHandler interface {
	// ExecContext executes a non-query SQL statement (e.g., INSERT, UPDATE, DELETE).
	ExecContext(ctx context.Context, statement Statement, param eval.Param) (sql.Result, error)

	// QueryContext executes a query SQL statement (e.g., SELECT).
	QueryContext(ctx context.Context, statement Statement, param eval.Param) (*sql.Rows, error)
}

// contextStatementHandler is a StatementHandler that wraps another StatementHandler
// and reduces the context with the session and parameter.
type contextStatementHandler struct {
	session session.Session
	next    StatementHandler
}

// QueryContext executes the query with the reduced context.
func (h *contextStatementHandler) QueryContext(ctx context.Context, statement Statement, param eval.Param) (*sql.Rows, error) {
	reducer := ctxreducer.G{
		ctxreducer.NewSessionContextReducer(h.session),
		ctxreducer.NewParamContextReducer(param),
	}
	ctx = reducer.Reduce(ctx)
	return h.next.QueryContext(ctx, statement, param)
}

// ExecContext executes the statement with the reduced context.
func (h *contextStatementHandler) ExecContext(ctx context.Context, statement Statement, param eval.Param) (sql.Result, error) {
	reducer := ctxreducer.G{
		ctxreducer.NewSessionContextReducer(h.session),
		ctxreducer.NewParamContextReducer(param),
	}
	ctx = reducer.Reduce(ctx)
	return h.next.ExecContext(ctx, statement, param)
}

// newContextStatementHandler creates a new contextStatementHandler.
func newContextStatementHandler(sess session.Session, next StatementHandler) StatementHandler {
	return &contextStatementHandler{session: sess, next: next}
}

// compiledStatementHandler executes pre-built SQL statements for performance.
type compiledStatementHandler struct {
	query         string
	args          []any
	middlewares   MiddlewareGroup
	driver        driver.Driver
	configuration Configuration
	queryHandler  QueryHandler
	execHandler   ExecHandler
}

// QueryContext executes a query and returns the resulting rows.
func (s *compiledStatementHandler) QueryContext(ctx context.Context, statement Statement, _ eval.Param) (*sql.Rows, error) {
	if s.queryHandler == nil {
		s.queryHandler = SessionQueryHandler
	}
	return s.middlewares.QueryContext(statement, s.configuration, s.queryHandler)(ctx, s.query, s.args...)
}

// ExecContext executes a non-query statement and returns the result.
func (s *compiledStatementHandler) ExecContext(ctx context.Context, statement Statement, _ eval.Param) (sql.Result, error) {
	if s.execHandler == nil {
		s.execHandler = SessionExecHandler
	}
	return s.middlewares.ExecContext(statement, s.configuration, s.execHandler)(ctx, s.query, s.args...)
}

// WithQueryHandler sets the QueryHandler for the compiledStatementHandler.
// This method is used to configure the handler that will be used to execute
// the compiled SQL query and return the resulting rows.
// It returns the same instance to allow for method chaining.
func (s *compiledStatementHandler) WithQueryHandler(queryHandler QueryHandler) *compiledStatementHandler {
	s.queryHandler = queryHandler
	return s
}

// WithExecHandler sets the ExecHandler for the compiledStatementHandler.
// This method is used to configure the handler that will be used to execute
// the compiled SQL statement that does not return rows (e.g., INSERT, UPDATE, DELETE).
// It returns the same instance to allow for method chaining.
func (s *compiledStatementHandler) WithExecHandler(execHandler ExecHandler) *compiledStatementHandler {
	s.execHandler = execHandler
	return s
}

// newCompiledStatementHandler creates a new instance of compiledStatementHandler with the provided
// query, arguments, middlewares, driver, session, and configuration.
// This is a private constructor function, intended for internal use within the package to
// create a pre-configured statement handler for executing compiled SQL statements.
// Optional handlers (QueryHandler, ExecHandler) can be set using the WithQueryHandler and
// WithExecHandler methods respectively.
func newCompiledStatementHandler(
	query string,
	args []any,
	middlewares MiddlewareGroup,
	driver driver.Driver,
	configuration Configuration,
) *compiledStatementHandler {
	handler := &compiledStatementHandler{
		query:         query,
		args:          args,
		middlewares:   middlewares,
		driver:        driver,
		configuration: configuration,
	}
	return handler
}

// buildStatementQuery builds the SQL query and its arguments from the given statement and parameters.
func buildStatementQuery(statement Statement, cfg Configuration, driver driver.Driver, param eval.Param) (string, []any, error) {
	parameter := buildStatementParameters(param, statement, driver.Name(), cfg)
	return statement.Build(driver.Translator(), parameter)
}

// preparedStatementHandler implements the StatementHandler interface.
// It maintains a single prepared statement that can be reused if the query is the same.
// When a different query is encountered, it closes the existing statement and creates a new one.
type preparedStatementHandler struct {
	stmts         *sql.Stmt
	middlewares   MiddlewareGroup
	driver        driver.Driver
	session       session.Session
	configuration Configuration
}

// getOrPrepare retrieves an existing prepared statement if the query matches,
// otherwise closes the current statement (if any) and creates a new one.
func (s *preparedStatementHandler) getOrPrepare(ctx context.Context, query string) (*sql.Stmt, error) {
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

// QueryContext executes a query that returns rows.
func (s *preparedStatementHandler) QueryContext(ctx context.Context, statement Statement, param eval.Param) (*sql.Rows, error) {
	query, args, err := buildStatementQuery(statement, s.configuration, s.driver, param)
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

	statementHandler := newCompiledStatementHandler(
		query,
		args,
		s.middlewares,
		s.driver,
		s.configuration,
	)
	statementHandler = statementHandler.WithQueryHandler(queryHandler)

	return newContextStatementHandler(s.session, statementHandler).QueryContext(ctx, statement, param)
}

// ExecContext executes a query that doesn't return rows.
func (s *preparedStatementHandler) ExecContext(ctx context.Context, statement Statement, param eval.Param) (result sql.Result, err error) {
	query, args, err := buildStatementQuery(statement, s.configuration, s.driver, param)
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

	statementHandler := newCompiledStatementHandler(
		query,
		args,
		s.middlewares,
		s.driver,
		s.configuration,
	)
	statementHandler = statementHandler.WithExecHandler(execHandler)

	return newContextStatementHandler(s.session, statementHandler).ExecContext(ctx, statement, param)
}

// Close closes all prepared statements in the pool and returns any error
// that occurred during the process. Multiple errors are joined together.
func (s *preparedStatementHandler) Close() error {
	if s.stmts != nil {
		return s.stmts.Close()
	}
	return nil
}

// newPreparedStatementHandler creates a new instance of preparedStatementHandler.
// This private constructor initializes the handler with the necessary dependencies
// for managing prepared statements, including middlewares, database driver,
// session, and configuration.
func newPreparedStatementHandler(
	middlewares MiddlewareGroup,
	driver driver.Driver,
	configuration Configuration,
	session session.Session,
) *preparedStatementHandler {
	return &preparedStatementHandler{
		middlewares:   middlewares,
		driver:        driver,
		session:       session,
		configuration: configuration,
	}
}

// queryBuildStatementHandler handles the execution of SQL statements and returns
// the results in a sql.Rows structure. It integrates a driver, middlewares, and
// a session to manage the execution flow.
type queryBuildStatementHandler struct {
	driver        driver.Driver
	middlewares   MiddlewareGroup
	session       session.Session
	configuration Configuration
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (s *queryBuildStatementHandler) QueryContext(ctx context.Context, statement Statement, param eval.Param) (*sql.Rows, error) {
	query, args, err := buildStatementQuery(statement, s.configuration, s.driver, param)
	if err != nil {
		return nil, err
	}

	statementHandler := newCompiledStatementHandler(
		query,
		args,
		s.middlewares,
		s.driver,
		s.configuration,
	)
	return newContextStatementHandler(s.session, statementHandler).QueryContext(ctx, statement, param)
}

// ExecContext executes a non-query SQL statement (such as INSERT, UPDATE, DELETE)
// within a context, and returns the result. Similar to QueryContext, it constructs
// the SQL command, applies middlewares, and executes the command using the driver.
func (s *queryBuildStatementHandler) ExecContext(ctx context.Context, statement Statement, param eval.Param) (sql.Result, error) {
	query, args, err := buildStatementQuery(statement, s.configuration, s.driver, param)
	if err != nil {
		return nil, err
	}

	statementHandler := newCompiledStatementHandler(
		query,
		args,
		s.middlewares,
		s.driver,
		s.configuration,
	)
	return newContextStatementHandler(s.session, statementHandler).ExecContext(ctx, statement, param)
}

var _ StatementHandler = (*queryBuildStatementHandler)(nil)

// newQueryBuildStatementHandler creates a new instance of queryBuildStatementHandler.
// This private constructor initializes the handler with the required dependencies
// for building and executing SQL statements, including the database driver,
// session, configuration, and an optional list of middlewares.
func newQueryBuildStatementHandler(
	driver driver.Driver,
	session session.Session,
	configuration Configuration,
	middlewares ...Middleware,
) *queryBuildStatementHandler {
	return &queryBuildStatementHandler{
		driver:        driver,
		middlewares:   middlewares,
		session:       session,
		configuration: configuration,
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
	driver        driver.Driver
	middlewares   MiddlewareGroup
	session       session.Session
	configuration Configuration
	value         reflect.Value
	batchSize     int64
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (s *sliceBatchStatementHandler) QueryContext(ctx context.Context, statement Statement, param eval.Param) (*sql.Rows, error) {
	statementHandler := newQueryBuildStatementHandler(s.driver, s.session, s.configuration, s.middlewares...)
	return statementHandler.QueryContext(ctx, statement, param)
}

func (s *sliceBatchStatementHandler) execContext(ctx context.Context, statement Statement, param eval.Param) (sql.Result, error) {
	statementHandler := newQueryBuildStatementHandler(s.driver, s.session, s.configuration, s.middlewares...)
	return statementHandler.ExecContext(ctx, statement, param)
}

func (s *sliceBatchStatementHandler) ExecContext(ctx context.Context, statement Statement, param eval.Param) (sql.Result, error) {
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
	preparedStmtHandler := newPreparedStatementHandler(s.middlewares, s.driver, s.configuration, s.session)

	// Ensure all prepared statements are properly closed after use
	defer func() { _ = preparedStmtHandler.Close() }()

	var batchErrs error
	aggregatedResult := &sqllib.BatchResult{}

	// execute the statement in batches.
	for i := 0; i < times; i++ {
		start := i * int(s.batchSize)
		end := (i + 1) * int(s.batchSize)
		if end > length {
			end = length
		}
		batchParam := s.value.Slice(start, end).Interface()
		result, err := preparedStmtHandler.ExecContext(ctx, statement, batchParam)
		if err != nil {
			if errors.Is(err, ErrBatchSkip) {
				batchErrs = errors.Join(batchErrs, err)
				continue
			}
			return nil, err
		}
		aggregatedResult.AccumulateResult(result)
	}

	if batchErrs != nil {
		return nil, batchErrs
	}
	return aggregatedResult, nil
}

// newSliceBatchStatementHandler creates a new instance of sliceBatchStatementHandler.
// This private constructor initializes the handler with the required dependencies
// for processing batch operations on slice parameters, including the database driver,
// middlewares, session, configuration, the slice value to process, and the batch size.
func newSliceBatchStatementHandler(
	driver driver.Driver,
	middlewares MiddlewareGroup,
	session session.Session,
	configuration Configuration,
	value reflect.Value,
	batchSize int64,
) *sliceBatchStatementHandler {
	return &sliceBatchStatementHandler{
		driver:        driver,
		middlewares:   middlewares,
		session:       session,
		configuration: configuration,
		value:         value,
		batchSize:     batchSize,
	}
}

type mapBatchStatementHandler struct {
	driver        driver.Driver
	middlewares   MiddlewareGroup
	configuration Configuration
	session       session.Session
	value         reflect.Value
	batchSize     int64
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (s *mapBatchStatementHandler) QueryContext(ctx context.Context, statement Statement, param eval.Param) (*sql.Rows, error) {
	statementHandler := newQueryBuildStatementHandler(s.driver, s.session, s.configuration, s.middlewares...)
	return statementHandler.QueryContext(ctx, statement, param)
}

func (s *mapBatchStatementHandler) execContext(ctx context.Context, statement Statement, param eval.Param) (sql.Result, error) {
	statementHandler := newQueryBuildStatementHandler(s.driver, s.session, s.configuration, s.middlewares...)
	return statementHandler.ExecContext(ctx, statement, param)
}

func (s *mapBatchStatementHandler) ExecContext(ctx context.Context, statement Statement, param eval.Param) (sql.Result, error) {
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
	preparedStmtHandler := newPreparedStatementHandler(s.middlewares, s.driver, s.configuration, s.session)

	// Ensure all prepared statements are properly closed after use
	defer func() { _ = preparedStmtHandler.Close() }()

	batchParam := reflect.MakeMap(s.value.Type())
	executionParam := batchParam.Interface()

	var batchErrs error
	aggregatedResult := &sqllib.BatchResult{}

	// execute the statement in batches.
	for i := 0; i < times; i++ {
		start := i * int(s.batchSize)
		end := (i + 1) * int(s.batchSize)
		if end > length {
			end = length
		}
		batchParam.SetMapIndex(keyValue, value.Slice(start, end))
		result, err := preparedStmtHandler.ExecContext(ctx, statement, executionParam)
		if err != nil {
			if errors.Is(err, ErrBatchSkip) {
				batchErrs = errors.Join(batchErrs, err)
				continue
			}
			return nil, err
		}
		aggregatedResult.AccumulateResult(result)
	}

	if batchErrs != nil {
		return nil, batchErrs
	}
	return aggregatedResult, nil
}

// newMapBatchStatementHandler creates a new instance of mapBatchStatementHandler.
// This private constructor initializes the handler with the required dependencies
// for processing batch operations on map parameters, including the database driver,
// middlewares, configuration, session, the map value to process, and the batch size.
func newMapBatchStatementHandler(
	driver driver.Driver,
	middlewares MiddlewareGroup,
	configuration Configuration,
	session session.Session,
	value reflect.Value,
	batchSize int64,
) *mapBatchStatementHandler {
	return &mapBatchStatementHandler{
		driver:        driver,
		middlewares:   middlewares,
		configuration: configuration,
		session:       session,
		value:         value,
		batchSize:     batchSize,
	}
}

// batchStatementHandler is a specialized SQL statement executor that provides optimized handling
// of batch operations, particularly for INSERT statements. It supports both single and batch
// execution modes, automatically switching to batch processing when:
// 1. The statement is an INSERT operation
// 2. A batch size is specified in the configuration
// 3. The input parameters represent multiple records (slice or map of structs)
//
// The handler integrates with the middleware chain and supports both regular and batch
// execution contexts. For non-batch operations, it behaves similarly to queryBuildStatementHandler.
type batchStatementHandler struct {
	driver        driver.Driver
	middlewares   MiddlewareGroup
	session       session.Session
	configuration Configuration
}

// QueryContext executes a query represented by the Statement object within a context,
// and returns the resulting rows. It builds the query using the provided Param values,
// processes the query through any configured middlewares, and then executes it using
// the associated driver.
func (b *batchStatementHandler) QueryContext(ctx context.Context, statement Statement, param eval.Param) (*sql.Rows, error) {
	statementHandler := newQueryBuildStatementHandler(b.driver, b.session, b.configuration, b.middlewares...)
	return statementHandler.QueryContext(ctx, statement, param)
}

// ExecContext executes a batch of SQL statements within a context. It handles
// the execution of SQL statements in batches if the action is an Insert and a
// batch size is specified. If the action is not an Insert or no batch size is
// specified, it delegates to the execContext method.
func (b *batchStatementHandler) ExecContext(ctx context.Context, statement Statement, param eval.Param) (result sql.Result, err error) {
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
		statementHandler = newSliceBatchStatementHandler(
			b.driver,
			b.middlewares,
			b.session,
			b.configuration,
			value.Unwrap().Value,
			batchSize,
		)
	case reflect.Map:
		statementHandler = newMapBatchStatementHandler(
			b.driver,
			b.middlewares,
			b.configuration,
			b.session,
			value.Unwrap().Value,
			batchSize,
		)
	default:
		return nil, errSliceOrArrayRequired
	}
	return statementHandler.ExecContext(ctx, statement, param)
}

func (b *batchStatementHandler) execContext(ctx context.Context, statement Statement, param eval.Param) (sql.Result, error) {
	statementHandler := newQueryBuildStatementHandler(b.driver, b.session, b.configuration, b.middlewares...)
	return statementHandler.ExecContext(ctx, statement, param)
}

// newBatchStatementHandler creates a new instance of batchStatementHandler.
// This private constructor initializes the handler with the required dependencies
// for processing batch operations, including the database driver, an optional list
// of middlewares, session, and configuration.
func newBatchStatementHandler(
	driver driver.Driver,
	session session.Session,
	configuration Configuration,
	middlewares ...Middleware,
) *batchStatementHandler {
	return &batchStatementHandler{
		driver:        driver,
		middlewares:   middlewares,
		session:       session,
		configuration: configuration,
	}
}
