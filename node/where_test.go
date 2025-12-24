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

func TestWhereNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node1 := NewTextNode("AND ID = #{ID}")
	node := NewTextNode("AND name = #{name}")
	node = WhereNode{
		Nodes: []Node{
			node1,
			node,
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
	if query != "WHERE ID = ? AND name = ?" {
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

func TestWhereNode_Accept_Comprehensive(t *testing.T) {
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
				&IfNode{Nodes: Group{NewTextNode("ID = #{ID}")}},
			},
			params:        emptyParams,
			expectedQuery: "",
			expectedArgs:  nil,
		},
		{
			name: "SingleCondition_NoLeadingAndOr",
			nodes: Group{
				NewTextNode("ID = #{ID}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectedQuery: "WHERE ID = ?",
			expectedArgs:  []any{1},
		},
		{
			name: "SingleCondition_LeadingAND",
			nodes: Group{
				NewTextNode("AND ID = #{ID}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectedQuery: "WHERE ID = ?",
			expectedArgs:  []any{1},
		},
		{
			name: "SingleCondition_LeadingOR",
			nodes: Group{
				NewTextNode("OR ID = #{ID}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectedQuery: "WHERE ID = ?",
			expectedArgs:  []any{1},
		},
		{
			name: "SingleCondition_LeadingLowercaseAND",
			nodes: Group{
				NewTextNode("and ID = #{ID}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectedQuery: "WHERE ID = ?",
			expectedArgs:  []any{1},
		},
		{
			name: "SingleCondition_LeadingLowercaseOR",
			nodes: Group{
				NewTextNode("or ID = #{ID}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectedQuery: "WHERE ID = ?",
			expectedArgs:  []any{1},
		},
		{
			name: "MultipleConditions_FirstNoLeading_SecondLeadingAND",
			nodes: Group{
				NewTextNode("status = #{status}"),
				NewTextNode("AND name = #{name}"),
			},
			params:        eval.NewGenericParam(eval.H{"status": "active", "name": "test"}, ""),
			expectedQuery: "WHERE status = ? AND name = ?",
			expectedArgs:  []any{"active", "test"},
		},
		{
			name: "MultipleConditions_FirstLeadingAND_SecondLeadingAND",
			nodes: Group{
				NewTextNode("AND status = #{status}"),
				NewTextNode("AND name = #{name}"),
			},
			params:        eval.NewGenericParam(eval.H{"status": "active", "name": "test"}, ""),
			expectedQuery: "WHERE status = ? AND name = ?",
			expectedArgs:  []any{"active", "test"},
		},
		{
			name: "QueryAlreadyStartsWithWHERE",
			nodes: Group{
				NewTextNode("WHERE ID = #{ID}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectedQuery: "WHERE ID = ?",
			expectedArgs:  []any{1},
		},
		{
			name: "QueryAlreadyStartsWithLowercaseWHERE",
			nodes: Group{
				NewTextNode("where ID = #{ID}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 1}, ""),
			expectedQuery: "where ID = ?",
			expectedArgs:  []any{1},
		},
		{
			name: "ChildNodeReturnsError",
			nodes: Group{
				&mockErrorNode{},
			},
			params:      emptyParams,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, n := range tt.nodes {
				if ifNode, ok := n.(*IfNode); ok {
					if tt.name == "ChildNodesProduceEmptyQuery" {
						if err := ifNode.Parse("1 == 0"); err != nil {
							t.Fatalf("Failed to parse IfNode condition for test %s: %v", tt.name, err)
						}
					} else {
						if ifNode.expr == nil {
							if parseErr := ifNode.Parse("true"); parseErr != nil {
								t.Logf("Default parsing for IfNode in test %s failed: %v", tt.name, parseErr)
							}
						}
					}
				}
			}

			node := WhereNode{Nodes: tt.nodes}
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
