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

import "github.com/go-juicedev/juice/sql"

// BindWithResultMap binds database query results to a value of type T using a custom ResultMap.
// This function provides backward compatibility for code that imports the juice package directly.
//
// For new code, consider using sql.BindWithResultMap directly:
//
//	import "github.com/go-juicedev/juice/sql"
//	user, err := sql.BindWithResultMap[User](rows, customResultMap)
//
// Parameters:
//   - rows: The database query result rows to bind from
//   - resultMap: Custom mapping strategy for converting rows to the target type
//
// Returns the bound value of type T and any error encountered during binding.
func BindWithResultMap[T any](rows sql.Rows, resultMap sql.ResultMap) (result T, err error) {
	return sql.BindWithResultMap[T](rows, resultMap)
}

// Bind converts database query results to a value of type T using the default mapping strategy.
// This function provides backward compatibility for code that imports the juice package directly.
//
// For new code, consider using sql.Bind directly:
//
//	import "github.com/go-juicedev/juice/sql"
//	user, err := sql.Bind[User](rows)
//
// The function automatically handles both single values and slices based on the type T.
// It uses struct field tags (default: "column") to map database columns to struct fields.
//
// Example:
//
//	type User struct {
//	    ID   int    `column:"id"`
//	    Name string `column:"name"`
//	}
//	user, err := Bind[User](rows)
//
// Returns the bound value of type T and any error encountered during binding.
func Bind[T any](rows sql.Rows) (result T, err error) {
	return sql.Bind[T](rows)
}

// List converts database query results to a slice of values of type T.
// This function provides backward compatibility for code that imports the juice package directly.
//
// For new code, consider using sql.List directly:
//
//	import "github.com/go-juicedev/juice/sql"
//	users, err := sql.List[User](rows)
//
// Unlike Bind, List always returns a slice []T, even for a single row.
// If there are no rows, it returns an empty slice (not nil, unless JUICE_RESULT_MAP_PRESERVE_NIL_SLICE is set).
//
// Example:
//
//	type User struct {
//	    ID   int    `column:"id"`
//	    Name string `column:"name"`
//	}
//	users, err := List[User](rows)  // Returns []User
//
// Returns a slice of values and any error encountered during binding.
func List[T any](rows sql.Rows) (result []T, err error) {
	return sql.List[T](rows)
}

// List2 converts database query results to a slice of pointers to values of type T.
// This function provides backward compatibility for code that imports the juice package directly.
//
// For new code, consider using sql.List2 directly:
//
//	import "github.com/go-juicedev/juice/sql"
//	users, err := sql.List2[User](rows)
//
// Unlike List which returns []T, List2 returns []*T. This is useful when:
//   - You need to modify slice elements after binding
//   - You're working with large structs and want to avoid copying
//   - You need to distinguish between zero values and missing data
//
// Example:
//
//	type User struct {
//	    ID   int    `column:"id"`
//	    Name string `column:"name"`
//	}
//	users, err := List2[User](rows)  // Returns []*User
//
// Returns a slice of pointers and any error encountered during binding.
func List2[T any](rows sql.Rows) ([]*T, error) {
	return sql.List2[T](rows)
}

// Iter creates an iterator for processing database query results row by row.
// This function provides backward compatibility for code that imports the juice package directly.
//
// For new code, consider using sql.Iter directly:
//
//	import "github.com/go-juicedev/juice/sql"
//	iter := sql.Iter[User](rows)
//
// The iterator implements Go's iter.Seq[T] interface, allowing use in range loops:
//
//	iter := Iter[User](rows)
//	for user := range iter.Iter() {
//	    // Process each user
//	    fmt.Println(user.Name)
//	}
//	if err := iter.Err(); err != nil {
//	    // Handle iteration error
//	}
//
// Note: The caller is responsible for closing the rows when iteration is complete.
//
// Returns an iterator that yields values of type T.
func Iter[T any](rows sql.Rows) *sql.RowsIter[T] {
	return sql.Iter[T](rows)
}
