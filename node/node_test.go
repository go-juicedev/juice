/*
Copyright 2023-2025 eatmoreapple

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

package node

import (
	"errors"
	"reflect"
	"testing"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

var errMock = errors.New("mock error")

type mockErrorNode struct{}

func (m *mockErrorNode) Accept(_ driver.Translator, _ eval.Parameter) (query string, args []any, err error) {
	return "", nil, errMock
}

type customStringer struct {
	val string
}

func (cs customStringer) String() string {
	return cs.val
}

type nonStringer struct {
	val int
}

func TestReflectValueToString_node_test(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"String", "hello", "hello"},
		{"EmptyString", "", ""},
		{"ByteSlice", []byte("world"), "world"},
		{"EmptyByteSlice", []byte{}, ""},
		{"FmtStringer", customStringer{"custom"}, "custom"},
		{"Int", 123, "123"},
		{"IntNegative", -456, "-456"},
		{"IntZero", 0, "0"},
		{"Int8", int8(12), "12"},
		{"Int16", int16(1234), "1234"},
		{"Int32", int32(123456), "123456"},
		{"Int64", int64(1234567890), "1234567890"},
		{"Uint", uint(789), "789"},
		{"UintZero", uint(0), "0"},
		{"Uint8", uint8(25), "25"},
		{"Uint16", uint16(5000), "5000"},
		{"Uint32", uint32(500000), "500000"},
		{"Uint64", uint64(9876543210), "9876543210"},
		{"Float32", float32(3.14), "3.14"},
		{"Float32Zero", float32(0.0), "0"},
		{"Float32Negative", float32(-2.71), "-2.71"},
		{"Float64", float64(2.71828), "2.71828"},
		{"Float64Zero", float64(0.0), "0"},
		{"Float64Negative", float64(-0.55), "-0.55"},
		{"BoolTrue", true, "true"},
		{"BoolFalse", false, "false"},
		{"PointerToString", new(string), ""},
		{"PointerToInt", func() *int { i := 10; return &i }(), "10"},
		{"InterfaceToString", interface{}("iface_string"), "iface_string"},
		{"InterfaceToInt", interface{}(42), "42"},
		{"StructNonStringer", nonStringer{100}, "{100}"},
		{"PointerToStructNonStringer", &nonStringer{200}, "{200}"},
	}

	sPtrVal := "pointed_string"
	sPtr := &sPtrVal
	tests = append(tests, struct {
		name     string
		input    any
		expected string
	}{"PointerToNonEmptyString", sPtr, "pointed_string"})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var val reflect.Value
			if tt.input == nil {
				var i interface{} = nil
				val = reflect.ValueOf(i)
			} else {
				val = reflect.ValueOf(tt.input)
			}

			result := reflectValueToString(val)
			if result != tt.expected {
				t.Errorf("reflectValueToString(%#v) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNodeGroup_Accept_node_test(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()
	emptyParams := eval.NewGenericParam(eval.H{}, "")

	t.Run("EmptyNodeGroup", func(t *testing.T) {
		ng := Group{}
		query, args, err := ng.Accept(translator, emptyParams)
		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
		if query != "" {
			t.Errorf("Expected empty query, but got %s", query)
		}
		if len(args) != 0 {
			t.Errorf("Expected no args, but got %v", args)
		}
	})

	t.Run("SingleNodeInGroup", func(t *testing.T) {
		ng := Group{
			NewTextNode("SELECT * FROM users WHERE ID = #{ID}"),
		}
		params := eval.NewGenericParam(eval.H{"ID": 1}, "")
		query, args, err := ng.Accept(translator, params)
		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
		if query != "SELECT * FROM users WHERE ID = ?" {
			t.Errorf("Expected 'SELECT * FROM users WHERE ID = ?', but got '%s'", query)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("Expected args [1], but got %v", args)
		}
	})

	t.Run("MultipleNodesInGroup", func(t *testing.T) {
		ng := Group{
			NewTextNode("SELECT *"),
			NewTextNode("FROM users"),
			NewTextNode("WHERE ID = #{ID}"),
		}
		params := eval.NewGenericParam(eval.H{"ID": 1}, "")
		query, args, err := ng.Accept(translator, params)
		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
		if query != "SELECT * FROM users WHERE ID = ?" {
			t.Errorf("Expected 'SELECT * FROM users WHERE ID = ?', but got '%s'", query)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("Expected args [1], but got %v", args)
		}
	})

	t.Run("MultipleNodesInGroupWithSpaces", func(t *testing.T) {
		ng := Group{
			NewTextNode("SELECT * "),
			NewTextNode(" FROM users "),
			NewTextNode(" WHERE ID = #{ID}"),
		}
		params := eval.NewGenericParam(eval.H{"ID": 1}, "")
		query, args, err := ng.Accept(translator, params)
		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
		if query != "SELECT *  FROM users  WHERE ID = ?" {
			t.Errorf("Expected 'SELECT *  FROM users  WHERE ID = ?', but got '%s'", query)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("Expected args [1], but got %v", args)
		}
	})

	t.Run("NodeReturnsError", func(t *testing.T) {
		errorNode := &mockErrorNode{}
		ng := Group{
			NewTextNode("SELECT *"),
			errorNode,
			NewTextNode("FROM users"),
		}
		params := eval.NewGenericParam(eval.H{"ID": 1}, "")
		_, _, err := ng.Accept(translator, params)
		if err == nil {
			t.Errorf("Expected an error, but got nil")
		}
		if err != errMock {
			t.Errorf("Expected errMock, but got %v", err)
		}
	})

	t.Run("NodeGroupWithEmptyAndNonEmptyNodes", func(t *testing.T) {
		ng := Group{
			NewTextNode(""),
			NewTextNode("SELECT * FROM table"),
			NewTextNode(""),
			NewTextNode("WHERE ID = #{ID}"),
		}
		params := eval.NewGenericParam(eval.H{"ID": 10}, "")
		query, args, err := ng.Accept(translator, params)
		if err != nil {
			t.Fatalf("Expected no error, but got %v", err)
		}
		expectedQuery := "SELECT * FROM table WHERE ID = ?"
		if query != expectedQuery {
			t.Errorf("Expected query '%s', but got '%s'", expectedQuery, query)
		}
		if len(args) != 1 || args[0] != 10 {
			t.Errorf("Expected args [10], but got %v", args)
		}
	})
}

// equalArgs is a helper to compare two slices of any.
func equalArgs(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper to parse expression and fail test if error.
func parseExprNoError(t *testing.T, exprStr string) eval.Expression {
	t.Helper()
	expr, err := eval.Compile(exprStr)
	if err != nil {
		t.Fatalf("Failed to parse expression '%s': %v", exprStr, err)
	}
	return expr
}

// mockMapper is a simplified mock for testing IncludeNode.
type mockMapper struct {
	nodes map[string]*SQLNode
	err   error
}

func (m *mockMapper) GetSQLNodeByID(id string) (*SQLNode, error) {
	if m.err != nil {
		return nil, m.err
	}
	node, exists := m.nodes[id]
	if !exists {
		return nil, errors.New("SQLNode with ID '" + id + "' not found in mockMapper")
	}
	return node, nil
}

// Ensure mockMapper implements the required part of the mapper for IncludeNode.
var _ interface {
	GetSQLNodeByID(id string) (*SQLNode, error)
} = (*mockMapper)(nil)
