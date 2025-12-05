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

package sqlmock

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

type MockRows struct {
	ColumnsLine     []string
	Data            [][]any
	ColumnsErr      error // Error to be returned by Columns()
	ScanErr         error // Error to be returned by Scan()
	Reason          error // Error to be returned by Err()
	nextErr         error // Error to be returned by Next() on a specific call
	forceNextReturn bool  // if true, Next() returns this value then false
	currentIndex    int
	closeCalled     bool
	nextReturnValue bool
}

func (m *MockRows) Columns() ([]string, error) {
	if m.ColumnsErr != nil {
		return nil, m.ColumnsErr
	}
	return m.ColumnsLine, nil
}

func (m *MockRows) Next() bool {
	if m.nextErr != nil {
		// Simulate error during iteration, Err() should pick this up
		m.Reason = m.nextErr
		return false
	}
	if m.forceNextReturn {
		ret := m.nextReturnValue
		m.forceNextReturn = false // only once
		return ret
	}
	if m.currentIndex < len(m.Data) {
		m.currentIndex++
		return true
	}
	return false
}

func (m *MockRows) Scan(dest ...any) error {
	if m.ScanErr != nil {
		return m.ScanErr
	}
	if m.currentIndex <= 0 || m.currentIndex > len(m.Data) {
		return errors.New("MockRows: Scan called out of bounds")
	}
	rowData := m.Data[m.currentIndex-1]
	if len(dest) != len(rowData) {
		return fmt.Errorf("MockRows: Scan expected %d dest args, got %d", len(rowData), len(dest))
	}
	for i, d := range dest {
		if scanner, ok := d.(sql.Scanner); ok {
			if err := scanner.Scan(rowData[i]); err != nil {
				return fmt.Errorf("MockRows: sql.Scanner Scan failed: %w", err)
			}
			continue
		}
		dv := reflect.ValueOf(d)
		if dv.Kind() != reflect.Ptr {
			return errors.New("MockRows: Scan destination not a pointer")
		}
		dv.Elem().Set(reflect.ValueOf(rowData[i]))
	}
	return nil
}

func (m *MockRows) Err() error {
	return m.Reason
}

func (m *MockRows) Close() error {
	m.closeCalled = true
	return nil
}
