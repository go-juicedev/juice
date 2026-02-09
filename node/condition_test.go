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

func TestConditionNode_Accept_condition_test(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()

	trueNode := NewTextNode("CONTENT_IF_TRUE")
	paramsTrue := eval.NewGenericParam(eval.H{"value": true, "number": 10, "text": "hello"}, "")
	paramsFalse := eval.NewGenericParam(eval.H{"value": false, "number": 0, "text": ""}, "")
	paramsError := eval.NewGenericParam(eval.H{"other": "value"}, "")

	tests := []struct {
		name             string
		condition        string
		params           eval.Parameter
		nodes            Group
		expectedQuery    string
		expectedArgs     []any
		expectError      bool
		parseShouldError bool
		matchShouldError bool
		expectedErrMsg   string
	}{
		{
			name:          "TrueCondition_Boolean",
			condition:     "value == true",
			params:        paramsTrue,
			nodes:         Group{trueNode},
			expectedQuery: "CONTENT_IF_TRUE",
		},
		{
			name:          "FalseCondition_Boolean",
			condition:     "value == false",
			params:        paramsTrue,
			nodes:         Group{trueNode},
			expectedQuery: "",
		},
		{
			name:          "TrueCondition_NumberNonZero",
			condition:     "number != 0",
			params:        paramsTrue,
			nodes:         Group{trueNode},
			expectedQuery: "CONTENT_IF_TRUE",
		},
		{
			name:          "FalseCondition_NumberZero",
			condition:     "number == 0",
			params:        paramsTrue,
			nodes:         Group{trueNode},
			expectedQuery: "",
		},
		{
			name:          "TrueCondition_StringNonEmpty",
			condition:     `text != ""`,
			params:        paramsTrue,
			nodes:         Group{trueNode},
			expectedQuery: "CONTENT_IF_TRUE",
		},
		{
			name:          "FalseCondition_StringEmpty",
			condition:     `text == ""`,
			params:        paramsTrue,
			nodes:         Group{trueNode},
			expectedQuery: "",
		},
		{
			name:          "FalseCondition_Boolean_WithFalseParam",
			condition:     "value == true",
			params:        paramsFalse,
			nodes:         Group{trueNode},
			expectedQuery: "",
		},
		{
			name:          "TrueCondition_NumberZero_WithFalseParam",
			condition:     "number == 0",
			params:        paramsFalse,
			nodes:         Group{trueNode},
			expectedQuery: "CONTENT_IF_TRUE",
		},
		{
			name:          "TrueCondition_StringEmpty_WithFalseParam",
			condition:     `text == ""`,
			params:        paramsFalse,
			nodes:         Group{trueNode},
			expectedQuery: "CONTENT_IF_TRUE",
		},
		{
			name:             "ParseError_InvalidExpression",
			condition:        "a b c",
			params:           paramsTrue,
			nodes:            Group{trueNode},
			expectError:      true,
			parseShouldError: true,
			expectedErrMsg:   "syntax error: 1:3: expected 'EOF', found b",
		},
		{
			name:             "MatchError_ParamNotFound",
			condition:        "missing_param == true",
			params:           paramsError,
			nodes:            Group{trueNode},
			expectError:      true,
			matchShouldError: true,
			expectedErrMsg:   "undefined identifier: missing_param",
		},
		{
			name:          "NoNodes",
			condition:     "value == true",
			params:        paramsTrue,
			nodes:         Group{},
			expectedQuery: "",
		},
		{
			name:           "NodeReturnsError",
			condition:      "value == true",
			params:         paramsTrue,
			nodes:          Group{&mockErrorNode{}},
			expectError:    true,
			expectedErrMsg: "mock error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ConditionNode{Nodes: tt.nodes}
			err := node.Parse(tt.condition)

			if tt.parseShouldError {
				if err == nil {
					t.Errorf("Expected a parse error, but got nil")
					return
				}
				if err.Error() != tt.expectedErrMsg {
					t.Errorf("Expected parse error message '%s', but got '%s'", tt.expectedErrMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			query, args, err := node.Accept(translator, tt.params)

			if tt.expectError && !tt.parseShouldError {
				if err == nil {
					t.Errorf("Expected an error from Accept/Match, but got nil. Query: %s", query)
					return
				}
				if tt.matchShouldError {
					if err.Error() != tt.expectedErrMsg {
						t.Errorf("Expected match error message '%s', but got '%s'", tt.expectedErrMsg, err.Error())
					}
				} else if err.Error() != tt.expectedErrMsg {
					t.Errorf("Expected accept error message '%s', but got '%s'", tt.expectedErrMsg, err.Error())
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
