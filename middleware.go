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
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"slices"
	"strconv"
	"time"

	"github.com/go-juicedev/juice/internal/reflectlite"
	"github.com/go-juicedev/juice/session"
)

const (
	// RandomDataSource selects a random datasource from all available sources
	RandomDataSource = "?"
	// RandomSecondaryDataSource selects a random datasource excluding the primary source
	RandomSecondaryDataSource = "?!"
)

// Middleware is a wrapper of QueryHandler and ExecHandler.
type Middleware interface {
	// QueryContext wraps the QueryHandler.
	QueryContext(stmt Statement, next QueryHandler) QueryHandler
	// ExecContext wraps the ExecHandler.
	ExecContext(stmt Statement, next ExecHandler) ExecHandler
}

// ensure MiddlewareGroup implements Middleware.
var _ Middleware = MiddlewareGroup(nil) // compile time check

// MiddlewareGroup is a group of Middleware.
type MiddlewareGroup []Middleware

// QueryContext implements Middleware.
// Call QueryContext will call all the QueryContext of the middlewares in the group.
func (m MiddlewareGroup) QueryContext(stmt Statement, next QueryHandler) QueryHandler {
	if len(m) == 0 {
		return next
	}
	for _, middleware := range m {
		next = middleware.QueryContext(stmt, next)
	}
	return next
}

// ExecContext implements Middleware.
// Call ExecContext will call all the ExecContext of the middlewares in the group.
func (m MiddlewareGroup) ExecContext(stmt Statement, next ExecHandler) ExecHandler {
	if len(m) == 0 {
		return next
	}
	for _, middleware := range m {
		next = middleware.ExecContext(stmt, next)
	}
	return next
}

// logger is a default logger for debug.
var logger = log.New(log.Writer(), "[juice] ", log.Flags())

// ensure DebugMiddleware implements Middleware.
var _ Middleware = (*DebugMiddleware)(nil) // compile time check

// DebugMiddleware is a middleware that prints the sql xmlSQLStatement and the execution time.
type DebugMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will print the sql xmlSQLStatement and the execution time.
func (m *DebugMiddleware) QueryContext(stmt Statement, next QueryHandler) QueryHandler {
	if !m.isDeBugMode(stmt) {
		return next
	}
	// wrapper QueryHandler
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[38m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Name(), query, args, spent)
		return rows, err
	}
}

// ExecContext implements Middleware.
// ExecContext will print the sql xmlSQLStatement and the execution time.
func (m *DebugMiddleware) ExecContext(stmt Statement, next ExecHandler) ExecHandler {
	if !m.isDeBugMode(stmt) {
		return next
	}
	// wrapper ExecContext
	return func(ctx context.Context, query string, args ...any) (sql.Result, error) {
		start := time.Now()
		rows, err := next(ctx, query, args...)
		spent := time.Since(start)
		logger.Printf("\x1b[33m[%s]\x1b[0m \x1b[32m %s\x1b[0m \x1b[38m %v\x1b[0m \x1b[31m %v\x1b[0m\n", stmt.Name(), query, args, spent)
		return rows, err
	}
}

// isDeBugMode returns true if the debug mode is on.
// Default debug mode is on.
// You can turn off the debug mode by setting the debug tag to false in the mapper xmlSQLStatement attribute or the configuration.
func (m *DebugMiddleware) isDeBugMode(stmt Statement) bool {
	// try to one the bug mode from the xmlSQLStatement
	debug := stmt.Attribute("debug")
	// if the bug mode is not set, try to one the bug mode from the Context
	if debug == "false" {
		return false
	}
	if cfg := stmt.Configuration(); cfg.Settings().Get("debug") == "false" {
		return false
	}
	return true
}

// ensure TimeoutMiddleware implements Middleware
var _ Middleware = (*TimeoutMiddleware)(nil) // compile time check

// TimeoutMiddleware is a middleware that sets the timeout for the sql xmlSQLStatement.
type TimeoutMiddleware struct{}

// QueryContext implements Middleware.
// QueryContext will set the timeout for the sql xmlSQLStatement.
func (t TimeoutMiddleware) QueryContext(stmt Statement, next QueryHandler) QueryHandler {
	timeout := t.getTimeout(stmt)
	if timeout <= 0 {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
		ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
		return next(ctx, query, args...)
	}
}

// ExecContext implements Middleware.
// ExecContext will set the timeout for the sql xmlSQLStatement.
func (t TimeoutMiddleware) ExecContext(stmt Statement, next ExecHandler) ExecHandler {
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

// getTimeout returns the timeout from the xmlSQLStatement.
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

// useGeneratedKeysMiddleware is a middleware that set the last insert id to the struct.
type useGeneratedKeysMiddleware struct{}

// QueryContext implements Middleware.
// return the result directly and do nothing.
func (m *useGeneratedKeysMiddleware) QueryContext(_ Statement, next QueryHandler) QueryHandler {
	return next
}

// ExecContext implements Middleware.
// ExecContext will set the last insert id to the struct.
func (m *useGeneratedKeysMiddleware) ExecContext(stmt Statement, next ExecHandler) ExecHandler {
	if !(stmt.Action() == Insert) {
		return next
	}
	const _useGeneratedKeys = "useGeneratedKeys"
	// If the useGeneratedKeys is not set or false, return the result directly.
	useGeneratedKeys := stmt.Attribute(_useGeneratedKeys) == "true" ||
		// If the useGeneratedKeys is not set, but the global useGeneratedKeys is set and true.
		stmt.Configuration().Settings().Get(_useGeneratedKeys) == "true"

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
		param := ParamFromContext(ctx)

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
	manager := ManagerFromContext(ctx)
	return IsTxManager(manager)
}

// TxSensitiveDataSourceSwitchMiddleware provides dynamic database routing capabilities
// while maintaining transaction safety. It supports explicit datasource naming,
// random selection from secondary sources (?), and random selection from all sources (!).
type TxSensitiveDataSourceSwitchMiddleware struct{}

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
	manager := ManagerFromContext(ctx)
	engine, ok := manager.(*Engine)
	if !ok {
		// In current implementation, this case should never happen.
		// But we keep this check as a safeguard for potential future changes.
		logger.Printf("[juice]: failed to switch datasource: %s, the manager is not an Engine", dataSourceName)
		return ctx, nil
	}
	chosenDataSourceName := t.chooseDataSourceName(dataSourceName, engine)
	if chosenDataSourceName == dataSourceName {
		return ctx, nil
	}
	db, _, err := engine.manager.Get(chosenDataSourceName)
	if err != nil {
		return nil, err
	}
	// inject the new session into the context
	return session.WithContext(ctx, db), nil
}

// QueryContext implements Middleware.QueryContext.
// It handles datasource switching for query operations while respecting transaction boundaries.
// The datasource is determined by the following priority:
// 1. Statement level 'dataSource' attribute
// 2. Global settings 'dataSource' configuration
// 3. Default to primary datasource if not configured
func (t *TxSensitiveDataSourceSwitchMiddleware) QueryContext(stmt Statement, next QueryHandler) QueryHandler {
	dataSource := stmt.Attribute("dataSource")
	if dataSource == "" {
		dataSource = stmt.Configuration().Settings().Get("selectDataSource").String()
	}
	if dataSource == "" {
		return next
	}
	return func(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
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

// ExecContext implements Middleware.ExecContext.
func (t *TxSensitiveDataSourceSwitchMiddleware) ExecContext(_ Statement, next ExecHandler) ExecHandler {
	return next
}
