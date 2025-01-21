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
	"database/sql"
	"errors"

	"github.com/go-juicedev/juice/driver"
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

func NewSQLRowsExecutor(statement Statement, statementHandler StatementHandler, driver driver.Driver) SQLRowsExecutor {
	return &sqlRowsExecutor{
		statement:        statement,
		statementHandler: statementHandler,
		driver:           driver,
	}
}

// ensure that the sqlRowsExecutor implements the SQLRowsExecutor interface.
var _ SQLRowsExecutor = (*sqlRowsExecutor)(nil)

// GenericExecutor is a generic sqlRowsExecutor.
type GenericExecutor[T any] struct {
	SQLRowsExecutor
}

// QueryContext executes the query and returns the scanner.
func (e *GenericExecutor[T]) QueryContext(ctx context.Context, p Param) (result T, err error) {
	// check the error of the sqlRowsExecutor
	if exe, ok := isInvalidExecutor(e.SQLRowsExecutor); ok {
		return result, exe.err
	}
	statement := e.Statement()

	retMap, err := statement.ResultMap()

	// ErrResultMapNotSet means the result map is not set, use the default result map.
	if err != nil {
		if !errors.Is(err, ErrResultMapNotSet) {
			return result, err
		}
	}

	// try to query the database.
	rows, err := e.SQLRowsExecutor.QueryContext(ctx, p)
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
	return e.SQLRowsExecutor.ExecContext(ctx, p)
}

// ensure GenericExecutor implements Executor.
var _ Executor[any] = (*GenericExecutor[any])(nil)
