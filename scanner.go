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

package juice

import (
	"database/sql"
	"reflect"
	"sync"
)

// RowScanner is an interface that provides a custom mechanism for mapping database rows
// to Go structures. It serves as an extension point in the data binding system,
// allowing implementers to override the default reflection-based mapping behavior.
//
// When a type implements this interface, the binding system will detect it during
// the mapping process and delegate the row scanning responsibility to the implementation.
// This gives complete control over how database values are mapped to the target structure.
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
//	func (u *User) ScanRows(rows *sql.Rows) error {
//	    return rows.Scan(&u.ID, &u.Name, &u.Email)
//	}
//
// The implementation must ensure proper handling of NULL values and return
// appropriate errors if the scanning process fails.
type RowScanner interface {
	ScanRows(rows *sql.Rows) error
}

// rowScannerType is the type of the RowScanner interface
var rowScannerType = reflect.TypeOf((*RowScanner)(nil)).Elem()

// rowScannerTypeImplementations is a thread-safe cache that stores information about
// whether types implement the RowScanner interface.
// It uses sync.Map instead of a regular map to avoid read/write conflicts in concurrent environments.
var rowScannerTypeImplementations = sync.Map{}

// isImplementsRowScanner checks if the given reflection type implements the RowScanner interface.
func isImplementsRowScanner(t reflect.Type) bool {
	value, exists := rowScannerTypeImplementations.Load(t)
	if exists {
		return value.(bool)
	}
	// this operation does not need to be atomic
	implements := t.Implements(rowScannerType)
	rowScannerTypeImplementations.Store(t, implements)
	return implements
}
