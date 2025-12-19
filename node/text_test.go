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
	"testing"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

func TestPureTextNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()
	emptyParams := eval.NewGenericParam(eval.H{}, "")

	tests := []struct {
		name     string
		nodeText string
		wantText string
	}{
		{
			name:     "SimpleText",
			nodeText: "SELECT 1",
			wantText: "SELECT 1",
		},
		{
			name:     "TextWithSpaces",
			nodeText: "  SELECT 1  ",
			wantText: "  SELECT 1  ",
		},
		{
			name:     "EmptyText",
			nodeText: "",
			wantText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := pureTextNode(tt.nodeText)
			query, args, err := node.Accept(translator, emptyParams)
			if err != nil {
				t.Errorf("pureTextNode.Accept() error = %v, wantErr nil", err)
				return
			}
			if query != tt.wantText {
				t.Errorf("pureTextNode.Accept() query = %v, want %v", query, tt.wantText)
			}
			if len(args) != 0 {
				t.Errorf("pureTextNode.Accept() args = %v, want empty", args)
			}
		})
	}
}

func TestTextNode_Accept_Comprehensive(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()

	tests := []struct {
		name           string
		text           string
		params         eval.Parameter
		expectedQuery  string
		expectedArgs   []any
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:          "NoPlaceholderNoSubstitution",
			text:          "SELECT * FROM users",
			params:        eval.NewGenericParam(eval.H{}, ""),
			expectedQuery: "SELECT * FROM users",
			expectedArgs:  nil,
		},
		{
			name:          "OnlyPlaceholder",
			text:          "SELECT * FROM users WHERE ID = #{ID} AND name = #{name}",
			params:        eval.NewGenericParam(eval.H{"ID": 1, "name": "Alice"}, ""),
			expectedQuery: "SELECT * FROM users WHERE ID = ? AND name = ?",
			expectedArgs:  []any{1, "Alice"},
		},
		{
			name:          "OnlySubstitution",
			text:          "SELECT * FROM ${tableName} WHERE status = '${status}'",
			params:        eval.NewGenericParam(eval.H{"tableName": "employees", "status": "active"}, ""),
			expectedQuery: "SELECT * FROM employees WHERE status = 'active'",
			expectedArgs:  nil,
		},
		{
			name:          "PlaceholderAndSubstitution",
			text:          "SELECT name FROM ${tableName} WHERE ID = #{ID} AND age > #{age}",
			params:        eval.NewGenericParam(eval.H{"tableName": "students", "ID": 101, "age": 20}, ""),
			expectedQuery: "SELECT name FROM students WHERE ID = ? AND age > ?",
			expectedArgs:  []any{101, 20},
		},
		{
			name:           "PlaceholderMissingParam",
			text:           "SELECT * FROM users WHERE ID = #{missing_id}",
			params:         eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectError:    true,
			expectedErrMsg: "parameter missing_id not found",
		},
		{
			name:           "SubstitutionMissingParam",
			text:           "SELECT * FROM ${missing_table}",
			params:         eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectError:    true,
			expectedErrMsg: "parameter missing_table not found",
		},
		{
			name:          "PlaceholderWithSpaces",
			text:          "SELECT * FROM users WHERE ID = #{  ID  }",
			params:        eval.NewGenericParam(eval.H{"ID": 5}, ""),
			expectedQuery: "SELECT * FROM users WHERE ID = ?",
			expectedArgs:  []any{5},
		},
		{
			name:          "SubstitutionWithSpaces",
			text:          "SELECT * FROM ${  tableName  }",
			params:        eval.NewGenericParam(eval.H{"tableName": "orders"}, ""),
			expectedQuery: "SELECT * FROM orders",
			expectedArgs:  nil,
		},
		{
			name:          "MultipleOccurrencesOfSamePlaceholder",
			text:          "SELECT #{ID}, name FROM users WHERE ID = #{ID}",
			params:        eval.NewGenericParam(eval.H{"ID": 7}, ""),
			expectedQuery: "SELECT ?, name FROM users WHERE ID = ?",
			expectedArgs:  []any{7, 7},
		},
		{
			name:          "MultipleOccurrencesOfSameSubstitution",
			text:          "SELECT ${column} FROM ${table} WHERE ${column} = 'test'",
			params:        eval.NewGenericParam(eval.H{"column": "data", "table": "items"}, ""),
			expectedQuery: "SELECT data FROM items WHERE data = 'test'",
			expectedArgs:  nil,
		},
		{
			name:          "PlaceholderWithDotNotation",
			text:          "SELECT * FROM users WHERE name = #{user.name}",
			params:        eval.NewGenericParam(eval.H{"user": map[string]any{"name": "Bob"}}, ""),
			expectedQuery: "SELECT * FROM users WHERE name = ?",
			expectedArgs:  []any{"Bob"},
		},
		{
			name:          "SubstitutionWithDotNotation",
			text:          "SELECT * FROM ${schema.table}",
			params:        eval.NewGenericParam(eval.H{"schema": map[string]any{"table": "public.users"}}, ""),
			expectedQuery: "SELECT * FROM public.users",
			expectedArgs:  nil,
		},
		{
			name:          "EmptyTextNode",
			text:          "",
			params:        eval.NewGenericParam(eval.H{}, ""),
			expectedQuery: "",
			expectedArgs:  nil,
		},
		{
			name:           "TextNodeWithOnlyPlaceholdersNoParams",
			text:           "ID = #{ID}",
			params:         eval.NewGenericParam(eval.H{}, ""),
			expectError:    true,
			expectedErrMsg: "parameter ID not found",
		},
		{
			name:           "TextNodeWithOnlySubstitutionsNoParams",
			text:           "TABLE ${table}",
			params:         eval.NewGenericParam(eval.H{}, ""),
			expectError:    true,
			expectedErrMsg: "parameter table not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewTextNode(tt.text)
			query, args, err := node.Accept(translator, tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil. Query: %s", query)
					return
				}
				if err.Error() != tt.expectedErrMsg {
					t.Errorf("Expected error message '%s', but got '%s'", tt.expectedErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, but got %v", err)
				return
			}

			if query != tt.expectedQuery {
				t.Errorf("Expected query '%s', but got '%s'", tt.expectedQuery, query)
			}

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, but got %d. Args: %v", len(tt.expectedArgs), len(args), args)
				return
			}

			for i, expectedArg := range tt.expectedArgs {
				if args[i] != expectedArg {
					t.Errorf("Expected arg %v at index %d, but got %v", expectedArg, i, args[i])
				}
			}
		})
	}
}

func TestTextNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node := NewTextNode("select * from user where ID = #{ID}")
	param := eval.NewGenericParam(eval.H{"ID": 1}, "")
	query, args, err := node.Accept(drv.Translator(), param)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "select * from user where ID = ?" {
		t.Error("query error")
		return
	}
	if len(args) != 1 {
		t.Error("args error")
		return
	}
	if args[0] != 1 {
		t.Error("args error")
		return
	}
}
