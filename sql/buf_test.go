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
	"database/sql"
	"errors"
	"testing"
)

func TestRowsBuffer_buf_test(t *testing.T) {
	columns := []string{"id", "name"}
	data := [][]any{
		{1, "alice"},
		{2, "bob"},
	}
	rb := NewRowsBuffer(columns, data)

	// Test Columns
	cols, err := rb.Columns()
	if err != nil {
		t.Fatalf("Columns error: %v", err)
	}
	if len(cols) != 2 || cols[0] != "id" || cols[1] != "name" {
		t.Errorf("expected [id name], got %v", cols)
	}

	// Test Next and Scan
	if !rb.Next() {
		t.Fatal("expected Next to return true")
	}
	var id int
	var name string
	if err := rb.Scan(&id, &name); err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if id != 1 || name != "alice" {
		t.Errorf("expected 1 alice, got %d %s", id, name)
	}

	if !rb.Next() {
		t.Fatal("expected Next to return true")
	}
	if err := rb.Scan(&id, &name); err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if id != 2 || name != "bob" {
		t.Errorf("expected 2 bob, got %d %s", id, name)
	}

	if rb.Next() {
		t.Fatal("expected Next to return false")
	}

	// Test Scan after end
	if err := rb.Scan(&id, &name); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected ErrNoRows, got %v", err)
	}

	// Test Close
	if err := rb.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}

	// Test behavior after Close
	if rb.Next() {
		t.Error("expected Next to return false after Close")
	}
	if _, err := rb.Columns(); !errors.Is(err, sql.ErrConnDone) {
		t.Errorf("expected ErrConnDone from Columns after Close, got %v", err)
	}
	if err := rb.Scan(&id, &name); !errors.Is(err, sql.ErrConnDone) {
		t.Errorf("expected ErrConnDone from Scan after Close, got %v", err)
	}
}

func TestRowsBuffer_ScanError_buf_test(t *testing.T) {
	columns := []string{"id"}
	data := [][]any{{1}}
	rb := NewRowsBuffer(columns, data)

	if !rb.Next() {
		t.Fatal("expected Next to return true")
	}

	var id int
	var name string
	err := rb.Scan(&id, &name)
	if err == nil {
		t.Fatal("expected error from Scan with wrong number of arguments")
	}
}
