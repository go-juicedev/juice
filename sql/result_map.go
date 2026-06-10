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

// Package sql provides a set of utilities for mapping database query results to Go data structures.
package sql

import (
	"cmp"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
)

var (
	// ErrTooManyRows is returned when a single-row mapping receives more than one row.
	ErrTooManyRows = errors.New("juice: too many rows in result set")
)

// ResultMap is an interface that defines a method for mapping database query results to Go data structures.
type ResultMap interface {
	// MapTo maps the data from the SQL row to the provided reflect.Value.
	MapTo(rv reflect.Value, row Rows) error
}

// SingleRowResultMap maps one SQL row to a non-slice destination.
type SingleRowResultMap struct{}

// MapTo implements ResultMapper interface.
// It maps data from a SQL row to the provided reflect.Value.
// If more than one row is returned from the query, it returns an ErrTooManyRows error.
func (SingleRowResultMap) MapTo(rv reflect.Value, rows Rows) error {
	// Validate input is a pointer
	if rv.Kind() != reflect.Pointer {
		return ErrPointerRequired
	}

	// Check if there is any row and handle potential errors
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("error occurred while fetching row: %w", err)
		}
		return sql.ErrNoRows
	}

	if rowScanner, ok := rv.Interface().(RowScanner); ok {
		if err := rowScanner.ScanRow(rows); err != nil {
			return fmt.Errorf("failed to scan row using RowScanner: %w", err)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("error occurred during row scanning: %w", err)
		}
		if rows.Next() {
			return ErrTooManyRows
		}
		return nil
	}

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Create destination mapper
	columnDest := &rowDestination{}

	// Map columns to struct fields and create scan destinations
	dest, err := columnDest.Destination(rv, columns)
	if err != nil {
		return fmt.Errorf("failed to create destination mapping: %w", err)
	}

	// Scan row data into destinations
	if err = rows.Scan(dest...); err != nil {
		return fmt.Errorf("failed to scan row: %w", err)
	}

	// Check for any errors that occurred during row scanning
	if err = rows.Err(); err != nil {
		return fmt.Errorf("error occurred during row scanning: %w", err)
	}

	// Ensure there is only one row
	if rows.Next() {
		return ErrTooManyRows
	}

	return nil
}

// resultMapPreserveNilSlice is a flag that indicates whether to preserve nil slices in the result map.
var resultMapPreserveNilSlice = os.Getenv("JUICE_RESULT_MAP_PRESERVE_NIL_SLICE") == "true"

// MultiRowsResultMap maps SQL rows to a slice destination.
type MultiRowsResultMap struct {
	New func() reflect.Value
}

// MapTo implements ResultMapper interface.
// It maps data from SQL rows to the provided reflect.Value.
// The reflect.Value must be a pointer to a slice.
// Each row is mapped to a new slice element.
func (m MultiRowsResultMap) MapTo(rv reflect.Value, rows Rows) error {
	if err := m.validateInput(rv); err != nil {
		return err
	}

	target := rv.Elem()

	elementType := target.Type().Elem()
	// get the element type and check if it's a pointer
	isPointer, isElementImplementsScanner := m.resolveTypes(elementType)

	// initialize element creator if not provided
	if m.New == nil {
		targetElementType := elementType
		if isPointer {
			targetElementType = targetElementType.Elem()
		}
		m.New = func() reflect.Value { return reflect.New(targetElementType) }
	}

	// map the rows to values
	values, err := m.mapRows(rows, isPointer, isElementImplementsScanner)
	if err != nil {
		return err
	}

	if len(values) > 0 {
		// Since we've already verified the type compatibility above,
		// we can safely grow the slice without additional type checks.
		target.Grow(len(values))

		target.Set(reflect.Append(target, values...))
	} else {
		// https://github.com/go-juicedev/juice/issues/437
		if !resultMapPreserveNilSlice {
			target.Set(reflect.MakeSlice(target.Type(), 0, 0))
		}
	}
	return nil
}

// validateInput validates that the input reflect.Value is a pointer to a slice
func (m MultiRowsResultMap) validateInput(rv reflect.Value) error {
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("%w: expected pointer to slice", ErrPointerRequired)
	}
	if rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("expected pointer to slice, got pointer to %v", rv.Elem().Kind())
	}
	return nil
}

// resolveTypes returns whether the slice element is a pointer and whether each
// element should be mapped through RowScanner.
//
// RowScanner is checked on the pointer form of the element type because custom
// scanners usually mutate the receiver, so implementations are normally declared
// with a pointer receiver, for example:
//
//	func (u *User) ScanRow(row Row) error
//
// MultiRowsResultMap also creates new elements with reflect.New, which returns a
// pointer value. Checking *T for a []T target lets []T and []*T both use the same
// RowScanner implementation on *T.
func (m MultiRowsResultMap) resolveTypes(elementType reflect.Type) (bool, bool) {
	isPointer := elementType.Kind() == reflect.Pointer
	pointerType := elementType
	if !isPointer {
		pointerType = reflect.PointerTo(elementType)
	}
	return isPointer, isImplementsRowScanner(pointerType)
}

// mapRows maps the rows to a slice of reflect.Values
func (m MultiRowsResultMap) mapRows(rows Rows, isPointer bool, useScanner bool) ([]reflect.Value, error) {
	if useScanner {
		return m.mapWithRowScanner(rows, isPointer)
	}
	return m.mapWithColumnDestination(rows, isPointer)
}

// mapWithRowScanner maps rows using the RowScanner interface
func (m MultiRowsResultMap) mapWithRowScanner(rows Rows, isPointer bool) ([]reflect.Value, error) {
	// Pre-allocate slice with an initial capacity
	values := make([]reflect.Value, 0, 8)

	for rows.Next() {
		// Create a new instance. Since RowScanner is implemented with pointer receiver,
		// we always create a pointer type and use it directly for scanning
		newValue := m.New()

		rowScanner, _ := reflect.TypeAssert[RowScanner](newValue)
		if err := rowScanner.ScanRow(rows); err != nil {
			return nil, fmt.Errorf("failed to scan row using RowScanner: %w", err)
		}

		if isPointer {
			values = append(values, newValue)
		} else {
			values = append(values, newValue.Elem())
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while iterating rows: %w", err)
	}

	return values, nil
}

// mapWithColumnDestination maps rows using column destination
func (m MultiRowsResultMap) mapWithColumnDestination(rows Rows, isPointer bool) ([]reflect.Value, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}
	columnDest := &rowDestination{}
	// Pre-allocate slice with an initial capacity
	values := make([]reflect.Value, 0, 8)

	for rows.Next() {
		// Create a new instance and get its underlying value for column mapping
		newValue := m.New()

		// Map database columns to struct fields and create scan destinations
		dest, err := columnDest.Destination(newValue, columns)
		if err != nil {
			return nil, fmt.Errorf("failed to get destination: %w", err)
		}

		// Scan the current row into the destinations
		if err = rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Append either the pointer or the value based on the target type
		if isPointer {
			values = append(values, newValue)
		} else {
			values = append(values, newValue.Elem())
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while iterating rows: %w", err)
	}

	return values, nil
}

// ColumnDestination builds scan destinations for a row.
type ColumnDestination interface {
	// Destination returns scan destinations for the given reflect.Value and columns.
	Destination(rv reflect.Value, column []string) ([]any, error)
}

// columnTagName is the tag name used to map database columns to struct fields.
var columnTagName = cmp.Or(os.Getenv("JUICE_COLUMN_TAG_NAME"), "column")

func ColumnTagName() string {
	return columnTagName
}

// rowDestination implements ColumnDestination interface for mapping SQL query results
// to struct fields. It handles the mapping between database columns and struct fields
// by maintaining the field indexes and managing unmapped columns.
type rowDestination struct {
	// indexes stores the mapping between column positions and struct field indexes.
	// Each element is a slice of integers representing the path to the struct field:
	// - Empty slice means the column has no corresponding struct field
	// - Single integer means direct field access
	// - Multiple integers represent nested struct field access
	indexes [][]int

	// sink is a discard slot for unmapped columns during scanning.
	// The value has no semantic meaning; rows.Scan only needs an addressable target
	// for columns that do not map to any field.
	sink any

	// dest is a slice of interface{} values used to store pointers to the target struct fields.
	// Each element in dest is a pointer used by database/sql to scan directly into a target field.
	//
	// - If a column has no corresponding struct field, the element is set to &s.sink (a discard variable).
	// - If a column maps to a struct field, the element is set to the address of that field.
	//
	// Example:
	//   For a struct with fields ID and Name, and columns "id" and "name":
	//   dest will be []any{&ID, &Name}.
	//
	// dest is reused across multiple scans to avoid repeated memory allocations.
	// Before each use, it is reset (e.g., using clear or manually setting elements to nil)
	// to ensure no stale pointers are left from previous scans.
	dest []any
}

// Destination returns scan destinations for the given reflect.Value and columns.
func (s *rowDestination) Destination(rv reflect.Value, columns []string) ([]any, error) {
	dest, err := s.destination(rv, columns)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func (s *rowDestination) destinationForOneColumn(rv reflect.Value, columns []string) ([]any, error) {
	// if type is time.Time or implements sql.Scanner, we can scan it directly
	if rv.Elem().Type() == timeType || rv.Type().Implements(scannerType) {
		return []any{rv.Interface()}, nil
	}

	if reflect.Indirect(rv).Kind() == reflect.Struct {
		return s.destinationForStruct(rv, columns)
	}
	// default behavior
	return []any{rv.Interface()}, nil
}

func (s *rowDestination) destination(rv reflect.Value, columns []string) ([]any, error) {
	if len(columns) == 1 {
		return s.destinationForOneColumn(rv, columns)
	}

	kind := reflect.Indirect(rv).Kind()
	if kind == reflect.Struct {
		return s.destinationForStruct(rv, columns)
	}
	return nil, fmt.Errorf("expected struct, but got %s", kind)
}

func (s *rowDestination) destinationForStruct(rv reflect.Value, columns []string) ([]any, error) {
	rv = reflect.Indirect(rv)
	if len(s.indexes) == 0 {
		s.setIndexes(rv, columns)
	}

	// initialize dest if it's nil or clear it
	if s.dest == nil {
		s.dest = make([]any, len(columns))
	} else {
		clear(s.dest)
	}

	for i, indexes := range s.indexes {
		if len(indexes) == 0 {
			s.dest[i] = &s.sink
		} else {
			field := rv.FieldByIndex(indexes)
			if !field.CanAddr() || !field.CanSet() {
				return nil, fmt.Errorf("column %q maps to an unexported or unsettable field", columns[i])
			}
			s.dest[i] = field.Addr().Interface()
		}
	}
	return s.dest, nil
}

// destIndexCacheKey identifies a column-to-field index mapping.
//
// The mapping depends on both the destination struct type and the exact set
// (and order) of result columns, so both are part of the key. reflect.Type is
// comparable and can be used directly as a map key field.
type destIndexCacheKey struct {
	typ        reflect.Type
	columnsKey string
}

// destIndexCache caches the column-to-field index mapping ([][]int) per
// (struct type, columns) combination.
//
// The mapping is derived purely from the struct's tags and the result columns,
// so it is identical for every query that returns the same columns into the
// same type. Building it requires a full reflection walk over the struct
// fields; caching it turns that per-query cost into a one-time cost.
//
// Entries are write-once and the stored [][]int is only ever read afterwards
// (see destinationForStruct), so sharing a single slice across goroutines is
// safe without copying. The number of entries is bounded by the number of
// distinct (type, column set) combinations an application uses — effectively
// the number of distinct SELECT projections — so it converges and does not
// grow with traffic, row count, or uptime.
var destIndexCache sync.Map // map[destIndexCacheKey][][]int

// loadDestIndexes returns the cached column-to-field index mapping for key,
// if one has been stored.
func loadDestIndexes(key destIndexCacheKey) ([][]int, bool) {
	cached, ok := destIndexCache.Load(key)
	if !ok {
		return nil, false
	}
	return cached.([][]int), true
}

// storeDestIndexes publishes the column-to-field index mapping for key so that
// subsequent queries with the same (type, columns) can reuse it. If two
// goroutines race on the same key, the last writer wins; both built equivalent,
// read-only slices, so the choice is irrelevant to correctness.
func storeDestIndexes(key destIndexCacheKey, indexes [][]int) {
	destIndexCache.Store(key, indexes)
}

// buildColumnsKey joins columns into a cache key. A NUL byte is used as the
// separator because it cannot appear in SQL column names, and each segment is
// length-prefixed so no two distinct column sets can ever encode to the same
// key (e.g. ["a","b\x00c"] vs ["a\x00b","c"]).
func buildColumnsKey(columns []string) string {
	var b strings.Builder
	size := 0
	for _, c := range columns {
		size += len(c) + 5 // worst-case length prefix + separator
	}
	b.Grow(size)
	for _, c := range columns {
		b.WriteString(strconv.Itoa(len(c)))
		b.WriteByte(':')
		b.WriteString(c)
		b.WriteByte(0)
	}
	return b.String()
}

// setIndexes maps result columns to struct field indexes, using a global cache
// keyed by (struct type, columns) to avoid repeating the reflection walk on
// every query.
func (s *rowDestination) setIndexes(rv reflect.Value, columns []string) {
	tp := rv.Type()
	key := destIndexCacheKey{typ: tp, columnsKey: buildColumnsKey(columns)}

	// Fast path: reuse the previously computed mapping. The cached slice is
	// read-only from here on, so it is safe to share.
	if cached, ok := loadDestIndexes(key); ok {
		s.indexes = cached
		return
	}

	s.indexes = make([][]int, len(columns))

	// columnIndex is a map to store the index of the column.
	columnIndex := make(map[string]int, len(columns))
	for i, column := range columns {
		columnIndex[column] = i
	}

	// walk into the struct
	s.findFromStruct(tp, columnIndex, nil)

	// Publish for reuse by subsequent queries.
	storeDestIndexes(key, s.indexes)
}

// findFromStruct finds matching field indexes in the struct type.
func (s *rowDestination) findFromStruct(tp reflect.Type, columnIndex map[string]int, walk []int) {

	// finished is a helper function to check if the indexes completed or not.
	finished := func() bool {
		return slices.IndexFunc(s.indexes, func(v []int) bool { return len(v) == 0 }) == -1
	}

	// walk into the struct
	for i := 0; i < tp.NumField(); i++ {
		// if we find all the columns destination, we can stop.
		if finished() {
			break
		}
		field := tp.Field(i)
		tag := field.Tag.Get(columnTagName)
		// if the tag is empty or "-", we can skip it.
		if skip := tag == "" && !field.Anonymous || tag == "-"; skip {
			continue
		}
		// if the field is anonymous and the type is struct, we can walk into it.
		if deepScan := field.Anonymous && field.Type.Kind() == reflect.Struct && len(tag) == 0; deepScan {
			s.findFromStruct(field.Type, columnIndex, append(append([]int(nil), walk...), i))
			continue
		}
		// find the index of the column
		index, ok := columnIndex[tag]
		if !ok {
			continue
		}
		// set the index
		s.indexes[index] = append(walk, field.Index...)
	}
}
