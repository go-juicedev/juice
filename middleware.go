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

package juice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/internal/reflectlite"
	"github.com/go-juicedev/juice/session"
	"github.com/go-juicedev/juice/sql"
)

// StatementContext carries the execution metadata shared by a middleware chain.
// It keeps statement, parameter, engine, and session access explicit instead of
// requiring middleware to recover those values from context.Context.
// It is created by the engine for each statement execution.
type StatementContext struct {
	engine  *Engine
	stmt    Statement
	ctx     context.Context
	param   eval.Param
	session session.Session
}

// Engine returns the engine that owns the current statement execution.
func (m *StatementContext) Engine() *Engine { return m.engine }

// Statement returns the mapped statement being executed.
func (m *StatementContext) Statement() Statement { return m.stmt }

// Context returns the request context passed to the executor.
func (m *StatementContext) Context() context.Context { return m.ctx }

// Param returns the parameter used to build the mapped statement.
func (m *StatementContext) Param() eval.Param { return m.param }

// Session returns the session currently used by the execution chain.
func (m *StatementContext) Session() session.Session { return m.session }

// WithSession replaces the session used by the execution chain.
func (m *StatementContext) WithSession(session session.Session) { m.session = session }

func newStatementContext(
	ctx context.Context,
	engine *Engine,
	stmt Statement,
	param eval.Param,
	session session.Session,
) *StatementContext {
	return &StatementContext{
		engine:  engine,
		stmt:    stmt,
		ctx:     ctx,
		param:   param,
		session: session,
	}
}

// Middleware defines the interface for intercepting and processing SQL executions.
// A middleware receives the current execution metadata and the next handler in the
// chain, then returns a handler that may run logic before, after, or instead of next.
type Middleware interface {
	// QueryContext intercepts and processes SELECT query executions.
	QueryContext(ctx *StatementContext, next QueryHandler) QueryHandler

	// ExecContext intercepts and processes INSERT/UPDATE/DELETE executions.
	ExecContext(ctx *StatementContext, next ExecHandler) ExecHandler
}

// ensure MiddlewareGroup implements Middleware.
var _ Middleware = MiddlewareGroup(nil) // compile time check

// MiddlewareGroup is a chain of middleware that implements the Middleware interface.
// It composes middlewares in registration order and returns one executable handler.
//
// Important: the last registered middleware becomes the outermost wrapper. If
// middlewares are registered as A, B, and C, composition is built as:
//
//	base handler
//	A(base handler)
//	B(A(base handler))
//	C(B(A(base handler)))
//
// Therefore the runtime flow is:
//
//	C before -> B before -> A before -> base handler -> A after -> B after -> C after
//
// Engine.Use appends to this group, so calling Use later means the middleware sees
// the request earlier and sees the response later.
type MiddlewareGroup []Middleware

// QueryContext implements Middleware.
// It composes the middleware chain for SELECT queries. The loop applies wrappers
// in slice order, so the last middleware in the slice is the first one executed
// at runtime.
// Returns a QueryHandler that executes the full middleware chain.
func (m MiddlewareGroup) QueryContext(ctx *StatementContext, next QueryHandler) QueryHandler {
	if len(m) == 0 {
		return next
	}
	for _, middleware := range m {
		next = middleware.QueryContext(ctx, next)
	}
	return next
}

// ExecContext implements Middleware.
// It composes the middleware chain for INSERT/UPDATE/DELETE executions. The loop
// applies wrappers in slice order, so the last middleware in the slice is the
// first one executed at runtime.
// Returns an ExecHandler that executes the full middleware chain.
func (m MiddlewareGroup) ExecContext(ctx *StatementContext, next ExecHandler) ExecHandler {
	if len(m) == 0 {
		return next
	}
	for _, middleware := range m {
		next = middleware.ExecContext(ctx, next)
	}
	return next
}

// NoopMiddleware is a middleware that performs no operations.
// It returns the original next handler.
type NoopMiddleware struct{}

// QueryContext implements Middleware.
func (n NoopMiddleware) QueryContext(_ *StatementContext, next QueryHandler) QueryHandler {
	return next
}

// ExecContext implements Middleware.
func (n NoopMiddleware) ExecContext(_ *StatementContext, next ExecHandler) ExecHandler {
	return next
}

// logger is a default logger for debug.
var logger = log.New(log.Writer(), "[juice] ", log.Flags())

// ensure DebugMiddleware implements Middleware.
var _ Middleware = (*DebugMiddleware)(nil) // compile time check

// DebugMiddleware is a middleware that logs SQL statements with their execution time and parameters.
// It provides debugging capabilities by printing formatted SQL queries along with execution metrics.
// The middleware can be enabled/disabled through statement attributes or global configuration settings.
type DebugMiddleware struct{}

func (m *DebugMiddleware) logRecord(id, query string, args []any, spent time.Duration) {
	// Ensure clean SQL presentation by removing trailing whitespace and formatting for optimal readability
	// while preserving the original SQL structure and parameter display
	if !strings.HasPrefix(query, "\n") {
		query = "\n" + query
	}
	query = strings.TrimRight(query, " \r\t\n")
	logger.Printf("\x1b[33m[%s]\x1b[0m args: \u001B[34m%v\u001B[0m time: \u001B[31m%v\u001B[0m \x1b[32m%s\x1b[0m",
		id, args, spent, query)
}

// QueryContext implements Middleware.
// QueryContext logs SQL SELECT statements with their execution time and parameters.
// The logging includes statement ID, SQL query, arguments, and execution duration.
// Logging is controlled by statement attributes or global debug settings.
func (m *DebugMiddleware) QueryContext(ctx *StatementContext, next QueryHandler) QueryHandler {
	stmt := ctx.Statement()
	if !m.isDeBugMode(stmt, ctx.Engine().GetConfiguration()) {
		return next
	}
	// wrapper QueryHandler
	return func(ctx context.Context, query string, args ...any) (sql.Rows, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		m.logRecord(stmt.Name(), query, args, spent)
		return rows, err
	}
}

// ExecContext implements Middleware.
// ExecContext logs SQL INSERT/UPDATE/DELETE statements with their execution time and parameters.
// The logging includes statement ID, SQL query, arguments, and execution duration.
// Logging is controlled by statement attributes or global debug settings.
func (m *DebugMiddleware) ExecContext(ctx *StatementContext, next ExecHandler) ExecHandler {
	stmt := ctx.Statement()
	if !m.isDeBugMode(stmt, ctx.Engine().GetConfiguration()) {
		return next
	}
	// wrapper ExecContext
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		m.logRecord(stmt.Name(), query, args, spent)
		return rows, err
	}
}

// isDeBugMode determines whether debug logging should be enabled for the statement.
// It checks debug settings in the following priority order:
// 1. Statement-level "debug" attribute (if set to "false", disables debug)
// 2. Global configuration "debug" setting (if set to "false", disables debug)
// 3. Default is true (debug mode enabled) if neither is explicitly set to false
//
// Returns true when debug logging is enabled.
func (m *DebugMiddleware) isDeBugMode(stmt Statement, configuration Configuration) bool {
	// Statement-level debug="false" disables logging.
	debug := stmt.Attribute("debug")
	if debug == "false" {
		return false
	}
	if configuration.Settings().Get("debug") == "false" {
		return false
	}
	return true
}

// ensure TimeoutMiddleware implements Middleware.
var _ Middleware = (*TimeoutMiddleware)(nil) // compile time check

// TimeoutMiddleware is a middleware that manages query execution timeouts.
// It sets context timeouts for SQL statements to prevent long-running queries from hanging.
// The timeout value is obtained from the statement's "timeout" attribute and is specified in milliseconds.
type TimeoutMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext sets a context timeout for SELECT queries to prevent long-running operations.
// The timeout value is obtained from the statement's "timeout" attribute.
// If timeout is <= 0, no timeout is applied and the original handler is returned unchanged.
func (t TimeoutMiddleware) QueryContext(ctx *StatementContext, next QueryHandler) QueryHandler {
	timeout := t.getTimeout(ctx.Statement())
	if timeout <= 0 {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Rows, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
		return next(ctx, query, args...)
	}
}

// ExecContext implements Middleware.
// ExecContext sets a context timeout for INSERT/UPDATE/DELETE operations to prevent long-running operations.
// The timeout value is obtained from the statement's "timeout" attribute.
// If timeout is <= 0, no timeout is applied and the original handler is returned unchanged.
func (t TimeoutMiddleware) ExecContext(ctx *StatementContext, next ExecHandler) ExecHandler {
	timeout := t.getTimeout(ctx.Statement())
	if timeout <= 0 {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
		return next(ctx, query, args...)
	}
}

// getTimeout retrieves the timeout value from the statement's "timeout" attribute.
// Returns the timeout value in milliseconds, or 0 if not set or invalid.
func (t TimeoutMiddleware) getTimeout(stmt Statement) (timeout int64) {
	timeoutAttr := stmt.Attribute("timeout")
	if timeoutAttr == "" {
		return
	}
	timeout, _ = strconv.ParseInt(timeoutAttr, 10, 64)
	return
}

// ensure useGeneratedKeysMiddleware implements Middleware
var _ Middleware = (*useGeneratedKeysMiddleware)(nil) // compile time check

// errStructPointerOrSliceArrayRequired is an error that the param is not a struct pointer or a slice array type.
var errStructPointerOrSliceArrayRequired = errors.New(
	"useGeneratedKeys is true, but the param is not a struct pointer or a slice array type",
)

// useGeneratedKeysMiddleware is a middleware that handles auto-generated primary keys for INSERT operations.
// It retrieves the last insert ID from the database result and sets it to the appropriate field in the parameter object.
// This middleware supports both single record and batch insert operations, with configurable key properties and increment strategies.
type useGeneratedKeysMiddleware struct {
	NoopMiddleware
}

// ExecContext implements Middleware.
// ExecContext processes INSERT operations to handle auto-generated primary keys.
// It retrieves the last insert ID from the database result and sets it to the appropriate field
// in the parameter object. Supports both single record and batch operations with configurable
// key properties and increment strategies.
func (m *useGeneratedKeysMiddleware) ExecContext(ctx *StatementContext, next ExecHandler) ExecHandler {
	stmt := ctx.Statement()

	if stmt.Action() != sql.Insert {
		return next
	}
	const _useGeneratedKeys = "useGeneratedKeys"
	// If the useGeneratedKeys is not set or false, return the result directly.
	// If the useGeneratedKeys is not set, but the global useGeneratedKeys is set and true.
	useGeneratedKeys := stmt.Attribute(_useGeneratedKeys) == "true" || ctx.Engine().GetConfiguration().Settings().Get(_useGeneratedKeys) == "true"

	if !useGeneratedKeys {
		return next
	}

	param := ctx.Param()

	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		result, err := next(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		// on most databases, the last insert ID is the first row affected.
		// calculate the last insert ID by the number of rows affected.
		if rowsAffected > 1 {
			id = id + rowsAffected - 1
		}

		// Support parameters wrapped in a single-entry map.
		rv := reflect.ValueOf(param)

		// A map wrapper must be unambiguous.
		if rv.Kind() == reflect.Map {
			if rv.Len() != 1 {
				return nil, fmt.Errorf("useGeneratedKeys is true, map must contain exactly one key-value pair, got %d", rv.Len())
			}
			// Extract the wrapped value.
			key := rv.MapKeys()[0]
			rv = rv.MapIndex(key)
		}

		// Unpack interface values before selecting a key generator.
		rv = reflectlite.Unpack(rv)

		keyProperty := stmt.Attribute("keyProperty")

		var keyGenerator selectKeyGenerator

		switch reflectlite.Unwrap(rv).Kind() {
		case reflect.Struct:
			keyGenerator = &singleKeyGenerator{
				keyProperty: keyProperty,
				id:          id,
			}
		case reflect.Array, reflect.Slice:
			// Use the configured key increment, or 1 when it is absent or invalid.
			keyIncrementValue := stmt.Attribute("keyIncrement")
			keyIncrement, _ := strconv.ParseInt(keyIncrementValue, 10, 64)
			keyIncrement = cmp.Or(keyIncrement, 1)
			// batchInsertIDGenerateStrategy is the strategy to generate the key in batch insert
			batchInsertIDStrategy := stmt.Attribute("batchInsertIDGenerateStrategy")
			keyGenerator = &batchKeyGenerator{
				keyProperty:                   keyProperty,
				id:                            id,
				keyIncrement:                  keyIncrement,
				batchInsertIDGenerateStrategy: batchInsertIDStrategy,
			}
		default:
			return nil, errStructPointerOrSliceArrayRequired
		}
		if err = keyGenerator.GenerateKeyTo(rv); err != nil {
			return nil, err
		}
		return result, nil
	}
}

// isInTransaction checks whether the active execution session is transactional.
func isInTransaction(sess session.Session) bool {
	_, ok := sess.(session.Transaction)
	return ok
}

const (
	// RandomDataSource selects a random datasource from all available sources
	RandomDataSource = "?"
	// RandomSecondaryDataSource selects a random datasource excluding the primary source
	RandomSecondaryDataSource = "?!"
)

// TxSensitiveDataSourceSwitchMiddleware provides dynamic database routing capabilities
// while maintaining transaction safety. It supports explicit datasource naming,
// random selection from secondary sources (?), and random selection from all sources (!).
//
// This middleware implements intelligent datasource switching based on:
// 1. Statement-level 'dataSource' attribute (highest priority)
// 2. Global 'selectDataSource' configuration setting
// 3. Transaction context awareness (avoids switching during transactions)
// 4. Support for random datasource selection strategies
//
// The middleware ensures that datasource switching only occurs outside of transactions
// to maintain data consistency and connection stability.
type TxSensitiveDataSourceSwitchMiddleware struct {
	NoopMiddleware
}

// selectRandomDataSource randomly selects a datasource from all available sources.
// If only one source is available, returns the current source.
func (t *TxSensitiveDataSourceSwitchMiddleware) selectRandomDataSource(engine *Engine) string {
	registeredEnvIds := engine.manager.Registered()
	if len(registeredEnvIds) == 1 {
		return engine.EnvID()
	}
	return registeredEnvIds[rand.Intn(len(registeredEnvIds))]
}

// selectRandomSecondaryDataSource randomly selects a datasource from secondary (non-primary) sources.
// If only primary source is available, returns the primary source.
func (t *TxSensitiveDataSourceSwitchMiddleware) selectRandomSecondaryDataSource(engine *Engine) string {
	registeredEnvIds := engine.manager.Registered()
	if len(registeredEnvIds) == 1 {
		return engine.EnvID()
	}

	var registeredEnvIdsReplica = make([]string, len(registeredEnvIds))
	copy(registeredEnvIdsReplica, registeredEnvIds)

	registeredEnvIdsReplica = slices.DeleteFunc(registeredEnvIdsReplica, func(envId string) bool {
		return envId == engine.EnvID()
	})

	if len(registeredEnvIdsReplica) == 0 {
		log.Printf("WARNING: No secondary data sources available after filtering, falling back to current engine: %s", engine.EnvID())
		return engine.EnvID()
	}

	return registeredEnvIdsReplica[rand.Intn(len(registeredEnvIdsReplica))]
}

// chooseDataSourceName selects the appropriate datasource based on the strategy:
// "?!" - random secondary source
// "?" - random from all sources
// otherwise - use the specified source
func (t *TxSensitiveDataSourceSwitchMiddleware) chooseDataSourceName(dataSourceName string, engine *Engine) string {
	switch dataSourceName {
	case RandomDataSource: // select a random source
		return t.selectRandomDataSource(engine)
	case RandomSecondaryDataSource: // ignore the primary source when selecting
		return t.selectRandomSecondaryDataSource(engine)
	default:
		return dataSourceName
	}
}

// switchDataSource updates the middleware session to use the selected datasource.
// It leaves the session unchanged when the selected datasource is already active.
func (t *TxSensitiveDataSourceSwitchMiddleware) switchDataSource(ctx *StatementContext, dataSourceName string) error {
	engine := ctx.Engine()
	chosenDataSourceName := t.chooseDataSourceName(dataSourceName, engine)

	// No switch is needed when the chosen datasource is already active.
	if chosenDataSourceName == engine.EnvID() {
		return nil
	}
	newEngine, err := engine.With(chosenDataSourceName)
	if err != nil {
		return err
	}

	// Route the remaining execution chain through the selected datasource session.
	ctx.WithSession(newEngine.DB())
	return nil
}

// QueryContext implements Middleware.
// QueryContext handles datasource switching for SELECT query operations while respecting transaction boundaries.
// The datasource is determined by the following priority:
// 1. Statement-level 'dataSource' attribute (highest priority)
// 2. Global 'selectDataSource' configuration setting
// 3. Default to primary datasource if not configured
//
// During transactions, datasource switching is disabled to maintain connection stability.
// Outside transactions, it can switch to alternative datasources based on configuration.
func (t *TxSensitiveDataSourceSwitchMiddleware) QueryContext(statementContext *StatementContext, next QueryHandler) QueryHandler {
	dataSource := statementContext.Statement().Attribute("dataSource")
	if dataSource == "" {
		dataSource = statementContext.Engine().GetConfiguration().Settings().Get("selectDataSource").String()
	}
	if dataSource == "" {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Rows, error) {
		if isInTransaction(statementContext.Session()) {
			return next(ctx, query, args...)
		}
		if err := t.switchDataSource(statementContext, dataSource); err != nil {
			return nil, err
		}
		return next(ctx, query, args...)
	}
}
