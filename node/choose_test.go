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
	"strings"
	"testing"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

func TestChooseNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()
	emptyParams := eval.NewGenericParam(eval.H{}, "")

	// Helper to create a ConditionNode (WhenNode) for testing
	newTestWhenNode := func(condition string, content string, paramsForParse eval.Parameter) *ConditionNode {
		cn := &ConditionNode{
			Nodes: Group{NewTextNode(content)},
		}
		err := cn.Parse(condition)
		if err != nil {
			panic("Failed to parse condition in test setup: " + err.Error())
		}
		return cn
	}

	// Helper to create an OtherwiseNode for testing
	newTestOtherwiseNode := func(content string) *OtherwiseNode {
		return &OtherwiseNode{
			Nodes: Group{NewTextNode(content)},
		}
	}

	paramsWithChoice := func(choice int) eval.Parameter {
		return eval.NewGenericParam(eval.H{"choice": choice, "name": "TestName"}, "")
	}

	errorWhenNode := &ConditionNode{Nodes: Group{&mockErrorNode{}}}
	if err := errorWhenNode.Parse("true"); err != nil {
		panic("Failed to parse condition for errorWhenNode: " + err.Error())
	}
	errorOtherwiseNode := &OtherwiseNode{Nodes: Group{&mockErrorNode{}}}

	tests := []struct {
		name           string
		whenNodes      []Node
		otherwiseNode  Node
		params         eval.Parameter
		expectedQuery  string
		expectedArgs   []any
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "FirstWhenMatches",
			whenNodes: []Node{
				newTestWhenNode("choice == 1", "Content for choice 1: #{name}", paramsWithChoice(1)),
				newTestWhenNode("choice == 2", "Content for choice 2", paramsWithChoice(1)),
			},
			otherwiseNode: newTestOtherwiseNode("Otherwise content"),
			params:        paramsWithChoice(1),
			expectedQuery: "Content for choice 1: ?",
			expectedArgs:  []any{"TestName"},
		},
		{
			name: "SecondWhenMatches",
			whenNodes: []Node{
				newTestWhenNode("choice == 1", "Content for choice 1", paramsWithChoice(2)),
				newTestWhenNode("choice == 2", "Content for choice 2: #{name}", paramsWithChoice(2)),
			},
			otherwiseNode: newTestOtherwiseNode("Otherwise content"),
			params:        paramsWithChoice(2),
			expectedQuery: "Content for choice 2: ?",
			expectedArgs:  []any{"TestName"},
		},
		{
			name: "NoWhenMatches_OtherwiseExecutes",
			whenNodes: []Node{
				newTestWhenNode("choice == 1", "Content for choice 1", paramsWithChoice(3)),
				newTestWhenNode("choice == 2", "Content for choice 2", paramsWithChoice(3)),
			},
			otherwiseNode: newTestOtherwiseNode("Otherwise content: #{name}"),
			params:        paramsWithChoice(3),
			expectedQuery: "Otherwise content: ?",
			expectedArgs:  []any{"TestName"},
		},
		{
			name: "NoWhenMatches_NoOtherwise",
			whenNodes: []Node{
				newTestWhenNode("choice == 1", "Content for choice 1", paramsWithChoice(3)),
				newTestWhenNode("choice == 2", "Content for choice 2", paramsWithChoice(3)),
			},
			params:        paramsWithChoice(3),
			expectedQuery: "",
		},
		{
			name: "WhenNodeItselfReturnsError",
			whenNodes: []Node{
				newTestWhenNode("choice == 0", "Should not be chosen", paramsWithChoice(1)),
				errorWhenNode,
				newTestWhenNode("choice == 2", "Should also not be chosen", paramsWithChoice(1)),
			},
			params:         paramsWithChoice(1),
			expectError:    true,
			expectedErrMsg: "mock error",
		},
		{
			name: "OtherwiseNodeReturnsError",
			whenNodes: []Node{
				newTestWhenNode("choice == 1", "Content for choice 1", paramsWithChoice(3)),
				newTestWhenNode("choice == 2", "Content for choice 2", paramsWithChoice(3)),
			},
			otherwiseNode:  errorOtherwiseNode,
			params:         paramsWithChoice(3),
			expectError:    true,
			expectedErrMsg: "mock error",
		},
		{
			name:          "EmptyWhenNodesList_OtherwiseExecutes",
			whenNodes:     []Node{},
			otherwiseNode: newTestOtherwiseNode("Only otherwise"),
			params:        emptyParams,
			expectedQuery: "Only otherwise",
		},
		{
			name: "OneWhenNode_NoMatch_NoOtherwise",
			whenNodes: []Node{
				newTestWhenNode("choice == 1", "Content for choice 1", paramsWithChoice(2)),
			},
			params:        paramsWithChoice(2),
			expectedQuery: "",
		},
		{
			name: "WhenNodeConditionParseError",
			whenNodes: []Node{
				func() Node {
					cn := &ConditionNode{Nodes: Group{NewTextNode("content")}}
					err := cn.Parse("invalid condition syntax @#$")
					if err == nil {
						panic("Expected parse error for test setup but got none")
					}
					return cn
				}(),
			},
			params:         emptyParams,
			expectError:    true,
			expectedErrMsg: ErrNilExpression.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := ChooseNode{
				WhenNodes:     tt.whenNodes,
				OtherwiseNode: tt.otherwiseNode,
			}
			query, args, err := node.Accept(translator, tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil. Query: %s", query)
					return
				}
				if !strings.Contains(err.Error(), tt.expectedErrMsg) {
					t.Errorf("Expected error message containing '%s', but got '%s'", tt.expectedErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, but got %v. Query: %s", err, query)
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
