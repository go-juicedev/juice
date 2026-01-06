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

const (
	// RandomDataSource selects a random datasource from all available sources
	RandomDataSource = "?"
	// RandomSecondaryDataSource selects a random datasource excluding the primary source
	RandomSecondaryDataSource = "?!"
)

// Middleware defines the interface for intercepting and processing SQL statement executions.
// It implements the interceptor pattern, allowing cross-cutting concerns like logging, 
// timeout management, and connection switching to be handled transparently.
type Middleware interface {
	// QueryContext intercepts and processes SELECT query executions.
	// It receives the statement, configuration, and the next handler in the chain.
	// Must return a QueryHandler that processes the actual query execution.
	QueryContext(stmt Statement, configuration Configuration, next QueryHandler) QueryHandler

	// ExecContext intercepts and processes INSERT/UPDATE/DELETE executions.
	// It receives the statement, configuration, and the next handler in the chain.
	// Must return an ExecHandler that processes the actual execution.
	ExecContext(stmt Statement, configuration Configuration, next ExecHandler) ExecHandler
}

// ensure MiddlewareGroup implements Middleware.
var _ Middleware = MiddlewareGroup(nil) // compile time check

// MiddlewareGroup is a chain of middleware that implements the Middleware interface.
// It executes middlewares in sequence, allowing multiple cross-cutting concerns to be
// applied to SQL statement execution. Each middleware can modify the behavior or add
// functionality before passing control to the next middleware in the chain.
type MiddlewareGroup []Middleware

// QueryContext implements Middleware.
// It processes the middleware chain for SELECT queries, executing each middleware in sequence.
// The last middleware in the chain will call the actual query handler (next parameter).
// Returns a QueryHandler that when called will execute the entire middleware chain.
func (m MiddlewareGroup) QueryContext(stmt Statement, configuration Configuration, next QueryHandler) QueryHandler {
	if len(m) == 0 {
		return next
	}
	for _, middleware := range m {
		next = middleware.QueryContext(stmt, configuration, next)
	}
	return next
}

// ExecContext implements Middleware.
// It processes the middleware chain for INSERT/UPDATE/DELETE executions, executing each middleware in sequence.
// The last middleware in the chain will call the actual execution handler (next parameter).
// Returns an ExecHandler that when called will execute the entire middleware chain.
func (m MiddlewareGroup) ExecContext(stmt Statement, configuration Configuration, next ExecHandler) ExecHandler {
	if len(m) == 0 {
		return next
	}
	for _, middleware := range m {
		next = middleware.ExecContext(stmt, configuration, next)
	}
	return next
}

// NoopQueryContextMiddleware is a middleware that performs no operations on QueryContext.
// It simply passes through the query execution without any modifications or interceptions.
// This middleware can be useful as a base implementation or placeholder when you need
// a middleware that doesn't affect query execution flow.
type NoopQueryContextMiddleware struct{}

// QueryContext implements Middleware interface.
// It returns the next handler in the chain without any modifications.
func (n NoopQueryContextMiddleware) QueryContext(_ Statement, _ Configuration, next QueryHandler) QueryHandler {
	return next
}

// ExecContext implements Middleware interface.
// It does nothing for ExecContext and returns the next handler as-is.
func (n NoopQueryContextMiddleware) ExecContext(_ Statement, _ Configuration, next ExecHandler) ExecHandler {
	panic("implement me")
}

// NoopExecContextMiddleware is a middleware that performs no operations on ExecContext.
// It simply passes through the execution without any modifications or interceptions.
// This middleware can be useful as a base implementation or placeholder when you need
// a middleware that doesn't affect execution flow.
type NoopExecContextMiddleware struct{}

// QueryContext implements Middleware interface.
// It does nothing for QueryContext and returns the next handler as-is.
func (n NoopExecContextMiddleware) QueryContext(_ Statement, _ Configuration, next QueryHandler) QueryHandler {
	panic("implement me")
}

// ExecContext implements Middleware interface.
// It returns the next handler in the chain without any modifications.
func (n NoopExecContextMiddleware) ExecContext(_ Statement, _ Configuration, next ExecHandler) ExecHandler {
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
// Logging is controlled by the debug mode setting from statement attributes or global configuration.
func (m *DebugMiddleware) QueryContext(stmt Statement, configuration Configuration, next QueryHandler) QueryHandler {
	if !m.isDeBugMode(stmt, configuration) {
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
// Logging is controlled by the debug mode setting from statement attributes or global configuration.
func (m *DebugMiddleware) ExecContext(stmt Statement, configuration Configuration, next ExecHandler) ExecHandler {
	if !m.isDeBugMode(stmt, configuration) {
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

// isDeBugMode determines whether debug logging should be enabled for the given statement.
// It checks debug settings in the following priority order:
// 1. Statement-level "debug" attribute (if set to "false", disables debug)
// 2. Global configuration "debug" setting (if set to "false", disables debug)
// 3. Default is true (debug mode enabled) if neither is explicitly set to false
//
// Returns true if debug mode should be enabled, false otherwise.
func (m *DebugMiddleware) isDeBugMode(stmt Statement, configuration Configuration) bool {
	// try to one the bug mode from the xmlSQLStatement
	debug := stmt.Attribute("debug")
	// if the bug mode is not set, try to one the bug mode from the Context
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
func (t TimeoutMiddleware) QueryContext(stmt Statement, _ Configuration, next QueryHandler) QueryHandler {
	timeout := t.getTimeout(stmt)
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
func (t TimeoutMiddleware) ExecContext(stmt Statement, _ Configuration, next ExecHandler) ExecHandler {
	timeout := t.getTimeout(stmt)
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
	NoopQueryContextMiddleware
}

// ExecContext implements Middleware.
// ExecContext processes INSERT operations to handle auto-generated primary keys.
// It retrieves the last insert ID from the database result and sets it to the appropriate field
// in the parameter object. Supports both single record and batch operations with configurable
// key properties and increment strategies.
func (m *useGeneratedKeysMiddleware) ExecContext(stmt Statement, configuration Configuration, next ExecHandler) ExecHandler {
	if stmt.Action() != sql.Insert {
		return next
	}
	const _useGeneratedKeys = "useGeneratedKeys"
	// If the useGeneratedKeys is not set or false, return the result directly.
	// If the useGeneratedKeys is not set, but the global useGeneratedKeys is set and true.
	useGeneratedKeys := stmt.Attribute(_useGeneratedKeys) == "true" || configuration.Settings().Get(_useGeneratedKeys) == "true"

	if !useGeneratedKeys {
		return next
	}
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
		// try to get param from context
		// ParamCtxInjectorExecutor is already set in middlewares, so the param should be in the context.
		param := eval.ParamFromContext(ctx)

		if param == nil {
			return nil, errors.New("useGeneratedKeys is true, but the param is nil")
		}

		// Handle special case where the input parameter might be wrapped in a map.
		// This allows for flexible parameter passing patterns, supporting both direct and wrapped formats.
		rv := reflect.ValueOf(param)

		// If the parameter is a map, we expect it to contain exactly one key-value pair
		// This restriction ensures unambiguous parameter extraction
		if rv.Kind() == reflect.Map {
			// Validate that the map contains exactly one entry
			// Multiple entries would create ambiguity about which value to use
			if rv.Len() != 1 {
				return nil, fmt.Errorf("useGeneratedKeys is true, map must contain exactly one key-value pair, got %d", rv.Len())
			}
			// Extract the single key and get its corresponding value
			// This value will be used for further processing
			key := rv.MapKeys()[0]
			rv = rv.MapIndex(key)
		}

		// unpack interface value
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
			// try to get the keyIncrement from the xmlSQLStatement
			// if the keyIncrement is not set or invalid, use the default value 1
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

// isInTransaction checks if the current context is within a transaction
func isInTransaction(ctx context.Context) bool {
	manager, _ := ManagerFromContext(ctx)
	return IsTxManager(manager)
}

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
	NoopExecContextMiddleware
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

// switchDataSource handles the datasource switching logic.
// It returns the original context if:
// - The manager is not an Engine
// - The chosen datasource is the same as the requested one
func (t *TxSensitiveDataSourceSwitchMiddleware) switchDataSource(ctx context.Context, dataSourceName string) (context.Context, error) {
	manager, _ := ManagerFromContext(ctx)
	engine, ok := manager.(*Engine)
	if !ok {
		// In current implementation, this case should never happen.
		// But we keep this check as a safeguard for potential future changes.
		logger.Printf("[juice]: failed to switch datasource: %s, the manager is not an Engine", dataSourceName)
		return ctx, nil
	}

	chosenDataSourceName := t.chooseDataSourceName(dataSourceName, engine)

	// no need to switch if the chosen datasource is the same as the current one
	if chosenDataSourceName == engine.EnvID() {
		return ctx, nil
	}

	newEngine, err := engine.With(chosenDataSourceName)
	if err != nil {
		return nil, err
	}

	// inject the new session into the context
	return session.WithContext(ctx, newEngine.DB()), nil
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
func (t *TxSensitiveDataSourceSwitchMiddleware) QueryContext(stmt Statement, configuration Configuration, next QueryHandler) QueryHandler {
	dataSource := stmt.Attribute("dataSource")
	if dataSource == "" {
		dataSource = configuration.Settings().Get("selectDataSource").String()
	}
	if dataSource == "" {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (sql.Rows, error) {
		if isInTransaction(ctx) {
			return next(ctx, query, args...)
		}
		ctx, err := t.switchDataSource(ctx, dataSource)
		if err != nil {
			return nil, err
		}
		return next(ctx, query, args...)
	}
}
