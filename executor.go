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
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/go-juicedev/juice/cache"
	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/internal/reflectlite"
)

// ErrInvalidExecutor is a custom error type that is used when an invalid executor is found.
var ErrInvalidExecutor = errors.New("juice: invalid executor")

// Executor is a generic sqlRowsExecutor.
type Executor[T any] interface {
	// QueryContext executes the query and returns the direct result.
	// The args are for any placeholder parameters in the query.
	QueryContext(ctx context.Context, param Param) (T, error)

	// ExecContext executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	ExecContext(ctx context.Context, param Param) (sql.Result, error)

	// Statement returns the Statement of the current Executor.
	Statement() Statement

	// Driver returns the driver of the current Executor.
	Driver() driver.Driver
}

// invalidExecutor wraps the error who implements the SQLRowsExecutor interface.
type invalidExecutor struct {
	_   struct{}
	err error
}

// QueryContext implements the SQLRowsExecutor interface.
func (b invalidExecutor) QueryContext(_ context.Context, _ Param) (*sql.Rows, error) {
	return nil, b.err
}

// ExecContext implements the SQLRowsExecutor interface.
func (b invalidExecutor) ExecContext(_ context.Context, _ Param) (sql.Result, error) {
	return nil, b.err
}

// Statement implements the SQLRowsExecutor interface.
func (b invalidExecutor) Statement() Statement { return nil }

func (b invalidExecutor) Driver() driver.Driver { return nil }

// SQLRowsExecutor defines the interface of the sqlRowsExecutor.
type SQLRowsExecutor Executor[*sql.Rows]

// inValidExecutor is an invalid sqlRowsExecutor.
func inValidExecutor(err error) SQLRowsExecutor {
	err = errors.Join(ErrInvalidExecutor, err)
	return &invalidExecutor{err: err}
}

// InValidExecutor returns an invalid sqlRowsExecutor.
func InValidExecutor() SQLRowsExecutor {
	return inValidExecutor(nil)
}

// isInvalidExecutor checks if the sqlRowsExecutor is a invalidExecutor.
func isInvalidExecutor(e SQLRowsExecutor) (*invalidExecutor, bool) {
	exe, ok := e.(*invalidExecutor)
	return exe, ok
}

// ensure that the defaultExecutor implements the SQLRowsExecutor interface.
var _ SQLRowsExecutor = (*invalidExecutor)(nil)

// sqlRowsExecutor implements the SQLRowsExecutor interface.
type sqlRowsExecutor struct {
	statement        Statement
	statementHandler StatementHandler
	driver           driver.Driver
}

// QueryContext executes the query and returns the result.
func (e *sqlRowsExecutor) QueryContext(ctx context.Context, param Param) (*sql.Rows, error) {
	return e.statementHandler.QueryContext(ctx, e.Statement(), param)
}

// ExecContext executes the query and returns the result.
func (e *sqlRowsExecutor) ExecContext(ctx context.Context, param Param) (sql.Result, error) {
	return e.statementHandler.ExecContext(ctx, e.Statement(), param)
}

// Statement returns the xmlSQLStatement.
func (e *sqlRowsExecutor) Statement() Statement { return e.statement }

// Driver returns the driver of the sqlRowsExecutor.
func (e *sqlRowsExecutor) Driver() driver.Driver { return e.driver }

// ensure that the sqlRowsExecutor implements the SQLRowsExecutor interface.
var _ SQLRowsExecutor = (*sqlRowsExecutor)(nil)

// cacheKeyFunc defines the function which is used to generate the scopeCache key.
type cacheKeyFunc func(stmt Statement, query string, args []any) (string, error)

// errCacheKeyFuncNil is an error that is returned when the CacheKeyFunc is nil.
var errCacheKeyFuncNil = errors.New("juice: CacheKeyFunc is nil")

// CacheKeyFunc is the function which is used to generate the scopeCache key.
// default is the md5 of the query and args.
// reset the CacheKeyFunc variable to change the default behavior.
var CacheKeyFunc cacheKeyFunc = func(stmt Statement, query string, args []any) (string, error) {
	// only same xmlSQLStatement same query same args can get the same scopeCache key
	writer := md5.New()
	writer.Write([]byte(stmt.ID() + query))
	if len(args) > 0 {
		if err := gob.NewEncoder(writer).Encode(args); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(writer.Sum(nil)), nil
}

// GenericExecutor is a generic sqlRowsExecutor.
type GenericExecutor[T any] struct {
	SQLRowsExecutor
	cache cache.ScopeCache
}

// QueryContext executes the query and returns the scanner.
func (e *GenericExecutor[T]) QueryContext(ctx context.Context, p Param) (result T, err error) {
	// check the error of the sqlRowsExecutor
	if exe, ok := isInvalidExecutor(e.SQLRowsExecutor); ok {
		return result, exe.err
	}
	statement := e.Statement()

	// if cache enabled
	cacheEnabled := e.cache != nil && statement.Attribute("useCache") != "false"

	if cacheEnabled {
		// cached this function in case the CacheKeyFunc is changed by other goroutines.
		keyFunc := CacheKeyFunc

		// check the keyFunc variable
		if keyFunc == nil {
			return result, errCacheKeyFuncNil
		}

		// build the query and args
		query, args, err := statement.Build(e.Driver().Translator(), p)
		if err != nil {
			return result, err
		}

		// get the type identify of the result
		typeIdentify := reflectlite.TypeIdentify[T]()
		// CacheKeyFunc is the function which is used to generate the scopeCache key.
		// default is the md5 of the query and args and the type identify.
		// reset the CacheKeyFunc variable to change the default behavior.
		cacheKey, err := keyFunc(statement, query+typeIdentify, args)
		if err != nil {
			return result, err
		}

		// try to get the result from the scopeCache
		if err = e.cache.Get(ctx, cacheKey, &result); err == nil {
			return result, err
		}
		// if we can not get the result from the scopeCache, continue with the next handler.
		if !errors.Is(err, cache.ErrCacheNotFound) {
			return result, err
		}

		// execute the query directly.
		result, err = e.queryContext(ctx, p)
		if err != nil {
			return result, err
		}
		// put the result to the scopeCache
		if err = e.cache.Set(ctx, cacheKey, result); err != nil {
			return result, fmt.Errorf("failed to set cache: %w", err)
		}
	} else {
		result, err = e.queryContext(ctx, p)
	}
	return
}

func (e *GenericExecutor[T]) queryContext(ctx context.Context, param Param) (result T, err error) {
	statement := e.Statement()

	retMap, err := statement.ResultMap()

	// ErrResultMapNotSet means the result map is not set, use the default result map.
	if err != nil {
		if !errors.Is(err, ErrResultMapNotSet) {
			return result, err
		}
	}

	// try to query the database.
	rows, err := e.SQLRowsExecutor.QueryContext(ctx, param)
	if err != nil {
		return result, err
	}
	defer func() { _ = rows.Close() }()

	return BindWithResultMap[T](rows, retMap)
}

// ExecContext executes the query and returns the result.
func (e *GenericExecutor[_]) ExecContext(ctx context.Context, p Param) (result sql.Result, err error) {
	// check the error of the sqlRowsExecutor
	if exe, ok := isInvalidExecutor(e.SQLRowsExecutor); ok {
		return nil, exe.err
	}
	result, err = e.SQLRowsExecutor.ExecContext(ctx, p)
	if err != nil {
		return
	}
	// if flushCache is true, flush the cache.
	if flushCache := e.cache != nil && e.Statement().Attribute("flushCache") != "false"; flushCache {
		err = e.cache.Flush(ctx)
	}
	return
}

// ensure GenericExecutor implements Executor.
var _ Executor[any] = (*GenericExecutor[any])(nil)
