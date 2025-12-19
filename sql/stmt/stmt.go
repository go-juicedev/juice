/*
Copyright 2024 eatmoreapple

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

package stmt

import (
	"database/sql"
	"reflect"
	"unsafe"
)

// queryFieldOffset stores the offset of the 'query' field in sql.Stmt struct
var queryFieldOffset uintptr

func init() {
	typ := reflect.TypeFor[sql.Stmt]()
	field, ok := typ.FieldByName("query")
	if !ok {
		panic("sql.Stmt structure has changed: 'query' field not found")
	}
	queryFieldOffset = field.Offset
}

// Query returns the underlying SQL query string from a *sql.Stmt.
// The offset of the query field is determined at init time using reflection,
// ensuring both safety and runtime performance.
//
// Note: This is an internal function that relies on the sql.Stmt structure.
// It will panic during package initialization if the structure changes.
func Query(s *sql.Stmt) string {
	return *(*string)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) + queryFieldOffset))
}

/* Implementation Notes:
This implementation uses reflection during initialization to safely obtain
the memory offset of the 'query' field in sql.Stmt. Once the offset is
determined, it uses unsafe.Pointer for efficient runtime access.

Benefits:
1. Safe: Field offset is obtained through reflection, not hardcoded
2. Fast: Runtime access uses direct pointer arithmetic
3. Maintainable: Will panic if sql.Stmt structure changes */
