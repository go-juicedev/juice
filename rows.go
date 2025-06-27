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

package juice

import "database/sql"

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type Rows interface {
	// Next prepares the next result row for reading with the Scan method. It
	// returns true on success, or false if there is no next result row or an error
	// happened while preparing it. Err should be consulted to distinguish between
	// the two cases.
	//
	// Every call to Scan, even the first one, must be preceded by a call to Next.
	Next() bool

	// Scan copies the columns in the current row into the values pointed at by dest.
	// The number of values in dest must be the same as the number of columns
	// in Rows.
	//
	// Scan converts columns read from the database into the following common
	// Go types and nil if the column value is NULL:
	//
	//    *string
	//    *[]byte
	//    *int, *int8, *int16, *int32, *int64
	//    *uint, *uint8, *uint16, *uint32, *uint64
	//    *bool
	//    *float32, *float64
	//    *interface{}
	//    *time.Time
	//
	// If a dest argument has type *[]byte, Scan saves in that argument a copy
	// of the corresponding data. The copy is owned by the caller and can be
	// modified and held indefinitely. The copy can be avoided by using an argument
	// of type *sql.RawBytes instead; see the documentation for sql.RawBytes for
	// details.
	//
	// If an error occurs during conversion, Scan returns the error.
	Scan(dest ...any) error

	// Close closes the Rows, preventing further enumeration. If Next is called
	// and returns false and there are no further result sets, the Rows are closed
	// automatically and it will suffice to check the result of Err. Close is
	// idempotent and does not affect the result of Err.
	Close() error

	// Err returns the error, if any, that was encountered during iteration.
	// Err may be called after an explicit or implicit Close.
	Err() error

	// Columns returns the column names.
	// Columns returns an error if the rows are closed.
	Columns() ([]string, error)
}

// var _ Rows = (*sql.Rows)(nil) ensures that *sql.Rows implements the Rows interface.
// This is a compile-time check and has no runtime overhead.
var _ Rows = (*sql.Rows)(nil)
