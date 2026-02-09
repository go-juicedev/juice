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

func TestSetNode_Accept_Comprehensive_set_test(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()
	emptyParams := eval.NewGenericParam(eval.H{}, "")

	tests := []struct {
		name          string
		nodes         Group
		params        eval.Parameter
		expectedQuery string
		expectedArgs  []any
		expectError   bool
	}{
		{
			name:          "EmptyChildNodes",
			nodes:         Group{},
			params:        emptyParams,
			expectedQuery: "",
			expectedArgs:  nil,
		},
		{
			name: "ChildNodesProduceEmptyQuery",
			nodes: Group{
				&IfNode{Nodes: Group{NewTextNode("ID = #{ID},")}},
			},
			params:        emptyParams,
			expectedQuery: "",
			expectedArgs:  nil,
		},
		{
			name: "SingleAssignment_NoTrailingComma",
			nodes: Group{
				NewTextNode("name = #{name}"),
			},
			params:        eval.NewGenericParam(eval.H{"name": "Test"}, ""),
			expectedQuery: "SET name = ?",
			expectedArgs:  []any{"Test"},
		},
		{
			name: "SingleAssignment_WithTrailingComma",
			nodes: Group{
				NewTextNode("name = #{name},"),
			},
			params:        eval.NewGenericParam(eval.H{"name": "Test"}, ""),
			expectedQuery: "SET name = ?",
			expectedArgs:  []any{"Test"},
		},
		{
			name: "MultipleAssignments_AllWithTrailingCommas",
			nodes: Group{
				NewTextNode("ID = #{ID},"),
				NewTextNode("name = #{name},"),
				NewTextNode("age = #{age},"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1, "name": "Test", "age": 30}, ""),
			expectedQuery: "SET ID = ?, name = ?, age = ?",
			expectedArgs:  []any{1, "Test", 30},
		},
		{
			name: "MultipleAssignments_LastWithoutTrailingComma",
			nodes: Group{
				NewTextNode("ID = #{ID},"),
				NewTextNode("name = #{name}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1, "name": "Test"}, ""),
			expectedQuery: "SET ID = ?, name = ?",
			expectedArgs:  []any{1, "Test"},
		},
		{
			name: "MultipleAssignments_MixedTrailingCommas",
			nodes: Group{
				NewTextNode("ID = #{ID}"),
				NewTextNode("name = #{name},"),
				NewTextNode("status = #{status}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1, "name": "Test", "status": "active"}, ""),
			expectedQuery: "SET ID = ? name = ?, status = ?",
			expectedArgs:  []any{1, "Test", "active"},
		},
		{
			name: "QueryAlreadyStartsWithSET",
			nodes: Group{
				NewTextNode("SET ID = #{ID}, name = #{name}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1, "name": "Test"}, ""),
			expectedQuery: "SET ID = ?, name = ?",
			expectedArgs:  []any{1, "Test"},
		},
		{
			name: "QueryAlreadyStartsWithLowercaseSet",
			nodes: Group{
				NewTextNode("set ID = #{ID}, name = #{name},"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1, "name": "Test"}, ""),
			expectedQuery: "set ID = ?, name = ?",
			expectedArgs:  []any{1, "Test"},
		},
		{
			name: "ChildNodeReturnsError",
			nodes: Group{
				&mockErrorNode{},
			},
			params:      emptyParams,
			expectError: true,
		},
		{
			name: "AssignmentsWithIfNodes",
			nodes: Group{
				&IfNode{Nodes: Group{NewTextNode("name = #{name},")}, expr: parseExprNoError(t, `name != ""`)},
				&IfNode{Nodes: Group{NewTextNode("age = #{age},")}, expr: parseExprNoError(t, "age > 0")},
				NewTextNode("modified_at = NOW()"),
			},
			params:        eval.NewGenericParam(eval.H{"name": "Valid Name", "age": 0}, ""),
			expectedQuery: "SET name = ?, modified_at = NOW()",
			expectedArgs:  []any{"Valid Name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "ChildNodesProduceEmptyQuery" {
				ifNode := tt.nodes[0].(*IfNode)
				if err := ifNode.Parse("1 == 0"); err != nil {
					t.Fatalf("Failed to parse IfNode condition for test %s: %v", tt.name, err)
				}
			}

			node := SetNode{Nodes: tt.nodes}
			query, args, err := node.Accept(translator, tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil. Query: %s", query)
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

			if !equalArgs(args, tt.expectedArgs) {
				t.Errorf("Expected args %v, but got %v", tt.expectedArgs, args)
			}
		})
	}
}

func TestSetNode_Accept_set_test(t *testing.T) {
	drv := driver.MySQLDriver{}
	node1 := NewTextNode("ID = #{ID},")
	node := NewTextNode("name = #{name},")
	node = SetNode{
		Nodes: []Node{
			node1, node,
		},
	}
	params := eval.H{
		"ID":   1,
		"name": "a",
	}
	query, args, err := node.Accept(drv.Translator(), eval.NewGenericParam(params, ""))
	if err != nil {
		t.Error(err)
		return
	}
	if query != "SET ID = ?, name = ?" {
		t.Error("query error")
		return
	}
	if len(args) != 2 {
		t.Error("args error")
		return
	}
	if args[0] != 1 || args[1] != "a" {
		t.Error("args error")
		return
	}
}
