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
	"database/sql"
	"errors"
	"iter"
	"reflect"
	"time"
)

var (
	// scannerType is the reflect.Type of sql.Scanner
	// nolint:unused
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

	// timeType is the reflect.Type of time.Time
	timeType = reflect.TypeOf((*time.Time)(nil)).Elem()
)

// bindWithResultMap maps Rows to a destination value using the specified ResultMap.
// It serves as the core binding function for all mapping operations in the package.
//
// Parameters:
//   - rows: The source Rows to map from. Must not be nil.
//   - v: The destination value to map to. Must be a pointer and not nil.
//   - resultMap: The mapping strategy to use. If nil, a default mapper will be selected
//     based on the destination type (SingleRowResultMap for struct, MultiRowsResultMap for slice).
//
// The function follows this process:
// 1. Validates input parameters
// 2. Checks if the destination implements RowScanner for custom mapping
// 3. Falls back to reflection-based mapping using the provided or default ResultMap
//
// Returns an error if:
//   - The destination is nil (ErrNilDestination)
//   - The rows parameter is nil (ErrNilRows)
//   - The destination is not a pointer (ErrPointerRequired)
//   - Any error occurs during the mapping process
func bindWithResultMap(rows Rows, v any, resultMap ResultMap) error {
	if v == nil {
		return ErrNilDestination
	}
	if rows == nil {
		return ErrNilRows
	}
	// Try custom row scanning if the destination implements RowScanner
	if rowScanner, ok := v.(RowScanner); ok {
		return rowScanner.ScanRows(rows)
	}
	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Ptr {
		return ErrPointerRequired
	}

	// Select default mapper if none provided
	if resultMap == nil {
		if kd := reflect.Indirect(rv).Kind(); kd == reflect.Slice {
			resultMap = MultiRowsResultMap{}
		} else {
			resultMap = SingleRowResultMap{}
		}
	}
	// Perform the actual mapping
	return resultMap.MapTo(rv, rows)
}

// BindWithResultMap bind Rows to given entity with given ResultMap
// bind cover Rows to given entity
// dest can be a pointer to a struct, a pointer to a slice of struct, or a pointer to a slice of any type.
// rows won't be closed when the function returns.
func BindWithResultMap[T any](rows Rows, resultMap ResultMap) (result T, err error) {
	// ptr is the pointer of the result, it is the destination of the binding.
	var ptr any = &result

	if _type := reflect.TypeOf(result); _type.Kind() == reflect.Ptr {
		// if the result is a pointer, create a new instance of the element.
		// you'd better not use a nil pointer as the result.
		result = reflect.New(_type.Elem()).Interface().(T)
		ptr = result
	}
	err = bindWithResultMap(rows, ptr, resultMap)
	return
}

// Bind Rows to given entity with default mapper
// Example usage of the binder package
//
// Example_bind shows how to use the Bind function:
//
//	type User struct {
//	    ID   int    `column:"id"`
//	    Name string `column:"name"`
//	}
//
//	rows, err := db.Query("SELECT id, name FROM users")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer rows.Close()
//
//	user, err := Bind[[]User](rows)
//	if err != nil {
//	    log.Fatal(err)
//	}
func Bind[T any](rows Rows) (result T, err error) {
	return BindWithResultMap[T](rows, nil)
}

// List converts Rows to a slice of the given entity type.
// If there are no rows, it will return an empty slice.
//
// Differences between List and Bind:
// - List always returns a slice, even if there is only one row.
// - Bind always returns the entity of the given type.
//
// Bind is more flexible; you can use it to bind a single row to a struct, a slice of structs, or a slice of any type.
// However, if you are sure that the result will be a slice, you can use List. It could be faster than Bind.
//
// Example_list shows how to use the List function:
//
//	type User struct {
//	    ID   int    `column:"id"`
//	    Name string `column:"name"`
//	}
//
//	rows, err := db.Query("SELECT id, name FROM users")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer rows.Close()
//
//	users, err := List[User](rows)
//	if err != nil {
//	    log.Fatal(err)
//	}
func List[T any](rows Rows) (result []T, err error) {
	var multiRowsResultMap MultiRowsResultMap

	element := reflect.TypeOf((*T)(nil)).Elem()

	// using reflect.New to create a new instance of the element is a very time-consuming operation.
	// if the element is not a pointer, we can create a new instance of it directly.
	if element.Kind() != reflect.Ptr {
		multiRowsResultMap.New = func() reflect.Value { return reflect.ValueOf(new(T)) }
	}

	err = bindWithResultMap(rows, &result, multiRowsResultMap)
	return
}

// List2 converts database query results into a slice of pointers.
// Unlike List function, List2 returns a slice of pointers []*T instead of a slice of values []T.
// This is particularly useful when you need to modify slice elements or handle large structs.
func List2[T any](rows Rows) ([]*T, error) {
	items, err := List[T](rows)
	if err != nil {
		return nil, err
	}
	var result = make([]*T, len(items))
	for i := range items {
		result[i] = &items[i]
	}
	return result, nil
}

// RowsIter provides an iterator interface for Rows.
// It implements Go's built-in iter.Seq interface for type-safe iteration over database rows.
// Type parameter T represents the type of values that will be yielded during iteration.
type RowsIter[T any] struct {
	rows Rows  // The underlying Rows to iterate over
	err  error // Stores any error that occurs during iteration
}

// Err returns any error that occurred during iteration.
// This method should be checked after iteration is complete to ensure
// no errors occurred while processing the rows.
func (r *RowsIter[T]) Err() error {
	return errors.Join(r.err, r.rows.Err())
}

// Iter implements the iter.Seq interface for row iteration.
// It yields values of type T, automatically handling memory allocation
// and type conversion for each row.
//
// Example usage:
//
//	iter := Iter[User](rows)
//	for v := range iter.Iter() {
//	    // Process each user
//	    fmt.Println(v.Name)
//	}
//	if err := iter.Err(); err != nil {
//	    // Handle error
//	}
func (r *RowsIter[T]) Iter() iter.Seq[T] {
	columns, err := r.rows.Columns()
	if err != nil {
		r.err = err
		return func(func(T) bool) {}
	}
	columnDest := &rowDestination{}
	t := reflect.TypeFor[T]()

	// Default object factory for non-pointer types
	var objectFactory = func() T { return *new(T) }

	isPtr := t.Kind() == reflect.Ptr

	// Override object factory for pointer types to properly allocate memory
	if isPtr {
		objectFactory = func() T { return reflect.New(t.Elem()).Interface().(T) }
	}

	// handler encapsulates the row scanning logic and object creation
	handler := func() (T, error) {
		var t = objectFactory()

		var v reflect.Value

		if isPtr {
			v = reflect.ValueOf(t)
		} else {
			v = reflect.ValueOf(&t)
		}

		// Create destination slice for scanning row values
		dest, err := columnDest.Destination(v, columns)
		if err != nil {
			return t, err
		}
		if err = r.rows.Scan(dest...); err != nil {
			return t, err
		}
		return t, nil
	}

	return func(yield func(T) bool) {

		for r.rows.Next() {
			value, err := handler()
			if err != nil {
				r.err = err
				return
			}
			if !yield(value) {
				return
			}
		}
	}
}

// Iter creates an iterator over SQL rows that yields values of type T.
// It handles both pointer and non-pointer types automatically and provides
// proper memory management for each iteration.
//
// Note: This function does not close the Rows. The caller is responsible
// for closing the rows when iteration is complete. This design allows for more
// flexible resource management, especially when using the iterator in different
// contexts or when early termination is needed.
func Iter[T any](rows Rows) *RowsIter[T] {
	return &RowsIter[T]{rows: rows}
}
