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

import "reflect"

// RowScanner provides a custom mechanism for mapping the current database row
// to a Go value. It serves as an extension point in the data binding system,
// allowing implementers to override the default reflection-based row mapping behavior.
//
// When a type implements this interface, the binding system will detect it during
// the mapping process and delegate the current row scanning responsibility to the
// implementation. ScanRow is called only after Rows.Next has successfully advanced
// the cursor, so implementations should call Row.Scan directly.
//
// Use cases:
// - Custom mapping logic for complex database schemas or legacy systems
// - Performance optimization by eliminating reflection overhead
// - Special data type handling (e.g., JSON, XML, custom database types)
// - Complex data transformations during the mapping process
// - Implementation of caching or lazy loading strategies
//
// Example implementation:
//
//	func (u *User) ScanRow(row Row) error {
//	    return row.Scan(&u.ID, &u.Name, &u.Email)
//	}
//
// The implementation must ensure proper handling of NULL values and return
// appropriate errors if the scanning process fails.
type RowScanner interface {
	ScanRow(row Row) error
}

var rowScannerType = reflect.TypeFor[RowScanner]()
