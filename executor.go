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
	"errors"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/sql"
)

// ErrInvalidExecutor marks an executor that could not be initialized.
var ErrInvalidExecutor = errors.New("juice: invalid executor")

// Executor executes SQL statements and returns typed query results.
type Executor[T any] interface {
	// QueryContext executes the query and returns the typed result.
	QueryContext(ctx context.Context, param eval.Param) (T, error)

	// ExecContext executes a statement that does not return rows.
	ExecContext(ctx context.Context, param eval.Param) (sql.Result, error)

	// Statement returns the mapped statement for this executor.
	Statement() Statement

	// Driver returns the driver of the current Executor.
	Driver() driver.Driver
}

// invalidExecutor stores an initialization error while satisfying SQLRowsExecutor.
type invalidExecutor struct {
	_   struct{}
	err error
}

// QueryContext implements the SQLRowsExecutor interface.
func (b invalidExecutor) QueryContext(_ context.Context, _ eval.Param) (sql.Rows, error) {
	return nil, b.err
}

// ExecContext implements the SQLRowsExecutor interface.
func (b invalidExecutor) ExecContext(_ context.Context, _ eval.Param) (sql.Result, error) {
	return nil, b.err
}

// Statement implements the SQLRowsExecutor interface.
func (b invalidExecutor) Statement() Statement { return nil }

func (b invalidExecutor) Driver() driver.Driver { return nil }

// SQLRowsExecutor is an Executor specialized for SQL rows.
type SQLRowsExecutor Executor[sql.Rows]

// inValidExecutor creates an executor that always returns err.
func inValidExecutor(err error) SQLRowsExecutor {
	err = errors.Join(ErrInvalidExecutor, err)
	return &invalidExecutor{err: err}
}

// InValidExecutor returns an executor that always fails.
func InValidExecutor() SQLRowsExecutor {
	return inValidExecutor(nil)
}

// isInvalidExecutor checks whether e is an invalidExecutor.
func isInvalidExecutor(e SQLRowsExecutor) (*invalidExecutor, bool) {
	exe, ok := e.(*invalidExecutor)
	return exe, ok
}

// ensure that the defaultExecutor implements the SQLRowsExecutor interface.
var _ SQLRowsExecutor = (*invalidExecutor)(nil)

// sqlRowsExecutor is the default SQLRowsExecutor implementation.
type sqlRowsExecutor struct {
	statement        Statement
	statementHandler StatementHandler
	driver           driver.Driver
}

// QueryContext executes the query and returns the result.
func (e *sqlRowsExecutor) QueryContext(ctx context.Context, param eval.Param) (sql.Rows, error) {
	return e.statementHandler.QueryContext(ctx, e.Statement(), param)
}

// ExecContext executes the query and returns the result.
func (e *sqlRowsExecutor) ExecContext(ctx context.Context, param eval.Param) (sql.Result, error) {
	return e.statementHandler.ExecContext(ctx, e.Statement(), param)
}

// Statement returns the mapped statement.
func (e *sqlRowsExecutor) Statement() Statement { return e.statement }

// Driver returns the executor's driver.
func (e *sqlRowsExecutor) Driver() driver.Driver { return e.driver }

func NewSQLRowsExecutor(statement Statement, statementHandler StatementHandler, driver driver.Driver) SQLRowsExecutor {
	return &sqlRowsExecutor{
		statement:        statement,
		statementHandler: statementHandler,
		driver:           driver,
	}
}

// Ensure sqlRowsExecutor implements SQLRowsExecutor.
var _ SQLRowsExecutor = (*sqlRowsExecutor)(nil)

// GenericExecutor binds SQL rows to a typed result.
type GenericExecutor[T any] struct {
	SQLRowsExecutor
}

// QueryContext executes the query and returns the scanner.
func (e *GenericExecutor[T]) QueryContext(ctx context.Context, p eval.Param) (result T, err error) {
	// Return deferred initialization errors before querying.
	if exe, ok := isInvalidExecutor(e.SQLRowsExecutor); ok {
		return result, exe.err
	}
	statement := e.Statement()

	retMap, err := statement.ResultMap()

	// ErrResultMapNotSet means the result map is not set, use the default result map.
	if err != nil {
		if !errors.Is(err, sql.ErrResultMapNotSet) {
			return result, err
		}
	}

	// try to query the database.
	rows, err := e.SQLRowsExecutor.QueryContext(ctx, p)
	if err != nil {
		return result, err
	}
	defer func() { _ = rows.Close() }()

	return sql.BindWithResultMap[T](rows, retMap)
}

// ExecContext executes the query and returns the result.
func (e *GenericExecutor[_]) ExecContext(ctx context.Context, p eval.Param) (result sql.Result, err error) {
	// Return deferred initialization errors before executing.
	if exe, ok := isInvalidExecutor(e.SQLRowsExecutor); ok {
		return nil, exe.err
	}
	return e.SQLRowsExecutor.ExecContext(ctx, p)
}

// ensure GenericExecutor implements Executor.
var _ Executor[any] = (*GenericExecutor[any])(nil)
