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

package sql

import (
	"database/sql"
	"fmt"
	_ "unsafe" // for go:linkname
)

// convertAssign is a linkname to the private convertAssign function in database/sql.
// It is used to perform high-performance, type-safe assignment of database-driver
// values to user-defined Go variables, following the same rules as sql.Rows.Scan.
//
//go:linkname convertAssign database/sql.convertAssign
func convertAssign(dest, src any) error

// RowsBuffer is a memory-based implementation of the Rows interface.
// It can be used to store query results in memory or for testing purposes.
type RowsBuffer struct {
	ColumnsLine []string
	Data        [][]any
	index       int // current index, 0 means before first row, 1 means first row
	closed      bool
}

// Columns returns the column names of the result set.
// It returns an error if the RowsBuffer is closed.
func (rb *RowsBuffer) Columns() ([]string, error) {
	if rb.closed {
		return nil, sql.ErrConnDone
	}
	return rb.ColumnsLine, nil
}

// Next advances the cursor to the next row of the result set.
// It returns false if there are no more rows or if the RowsBuffer is closed.
func (rb *RowsBuffer) Next() bool {
	if rb.closed {
		return false
	}
	rb.index++
	return rb.index <= len(rb.Data)
}

// Scan copies the columns in the current row into the values pointed at by dest.
// The number of values in dest must be the same as the number of columns in the row.
// It returns an error if the RowsBuffer is closed, if there are no rows at the current index,
// or if the number of destination arguments does not match the number of columns.
func (rb *RowsBuffer) Scan(dest ...any) error {
	if rb.closed {
		return sql.ErrConnDone
	}
	if rb.index <= 0 || rb.index > len(rb.Data) {
		return sql.ErrNoRows
	}
	row := rb.Data[rb.index-1]
	if len(dest) != len(row) {
		return fmt.Errorf("sql: expected %d destination arguments in Scan, not %d", len(row), len(dest))
	}
	for i := range dest {
		if err := convertAssign(dest[i], row[i]); err != nil {
			return err
		}
	}
	return nil
}

// Close marks the RowsBuffer as closed, preventing further operations.
func (rb *RowsBuffer) Close() error {
	rb.closed = true
	return nil
}

// Err returns the error encountered during iteration, if any.
// In this implementation, it always returns nil as the data is pre-buffered.
func (rb *RowsBuffer) Err() error {
	return nil
}

// NewRowsBuffer creates a new RowsBuffer with the given columns and data.
func NewRowsBuffer(columns []string, data [][]any) *RowsBuffer {
	return &RowsBuffer{
		ColumnsLine: columns,
		Data:        data,
		index:       0,
		closed:      false,
	}
}
