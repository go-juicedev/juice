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
	"iter"
	"reflect"
)

// Iterator is a type alias for iter.Seq2[T, error].
// It represents an iterator that yields values of type T and may return an error.
//
// This type is specifically designed for juicecli to use as a type annotation
// during code generation, allowing the tool to recognize and generate proper
// iterator-based query methods.
type Iterator[T any] iter.Seq2[T, error]

// Iter creates an iterator from the given Rows.
// It scans each row into a new instance of type T and yields it.
// If an error occurs during scanning, it yields the error.
func Iter[T any](rows Rows) (Iterator[T], error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnDest := &rowDestination{}
	t := reflect.TypeFor[T]()

	var objectFactory func() T

	isPtr := t.Kind() == reflect.Ptr

	// Override object factory for pointer types to properly allocate memory
	if isPtr {
		objectFactory = func() T {
			result, _ := reflect.TypeAssert[T](reflect.New(t.Elem()))
			return result
		}
	} else {
		objectFactory = func() T { return *new(T) }
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
		if err = rows.Scan(dest...); err != nil {
			return t, err
		}
		return t, nil
	}

	return func(yield func(T, error) bool) {
		for rows.Next() {
			value, err := handler()
			if !yield(value, err) {
				return
			}
		}
		// Check for any errors that occurred during iteration
		if err := rows.Err(); err != nil {
			var zero T
			yield(zero, err)
		}
	}, nil
}
