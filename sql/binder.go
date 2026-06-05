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

package sql

import (
	"database/sql"
	"reflect"
	"time"
)

var (
	// scannerType is the reflect.Type of sql.Scanner
	// nolint:unused
	scannerType = reflect.TypeFor[sql.Scanner]()

	// timeType is the reflect.Type of time.Time
	timeType = reflect.TypeFor[time.Time]()
)

// bindWithResultMap maps Rows into v using resultMap or a default mapper.
func bindWithResultMap(rows Rows, v any, resultMap ResultMap) error {
	if v == nil {
		return ErrNilDestination
	}
	if rows == nil {
		return ErrNilRows
	}
	rv := reflect.ValueOf(v)

	if rv.Kind() != reflect.Pointer {
		return ErrPointerRequired
	}

	// Select a default mapper when none is provided.
	if resultMap == nil {
		if kd := reflect.Indirect(rv).Kind(); kd == reflect.Slice {
			resultMap = MultiRowsResultMap{}
		} else {
			resultMap = SingleRowResultMap{}
		}
	}
	return resultMap.MapTo(rv, rows)
}

// BindWithResultMap binds Rows to T using resultMap.
// Rows is not closed by this function.
func BindWithResultMap[T any](rows Rows, resultMap ResultMap) (result T, err error) {
	// ptr is the destination used by the binding step.
	var ptr any = &result

	// For pointer result types, allocate the pointed-to value before scanning.
	if valueType := reflect.TypeFor[T](); valueType.Kind() == reflect.Pointer {
		result, _ = reflect.TypeAssert[T](reflect.New(valueType.Elem()))
		ptr = result
	}
	err = bindWithResultMap(rows, ptr, resultMap)
	return
}

// Bind maps Rows to T using the default mapper.
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
// If there are no rows, it returns an empty slice.
//
// Differences between List and Bind:
// - List always returns a slice, even if there is only one row.
// - Bind always returns the entity of the given type.
//
// Bind is more flexible; use List when the expected result is always a slice.
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

	element := reflect.TypeFor[T]()

	// Avoid reflect.New for non-pointer elements on the hot path.
	if element.Kind() != reflect.Pointer {
		multiRowsResultMap.New = func() reflect.Value { return reflect.ValueOf(new(T)) }
	}

	err = bindWithResultMap(rows, &result, multiRowsResultMap)
	return
}

// List2 converts Rows into []*T instead of []T.
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
