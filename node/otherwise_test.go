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

func TestOtherwiseNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()
	emptyParams := eval.NewGenericParam(eval.H{}, "")

	tests := []struct {
		name          string
		nodes         NodeGroup
		params        eval.Parameter
		expectedQuery string
		expectedArgs  []any
		expectError   bool
	}{
		{
			name:          "EmptyNodeGroup",
			nodes:         NodeGroup{},
			params:        emptyParams,
			expectedQuery: "",
			expectedArgs:  nil,
		},
		{
			name: "NodeGroupWithContent",
			nodes: NodeGroup{
				NewTextNode("DEFAULT CONTENT WHERE ID = #{ID}"),
			},
			params:        eval.NewGenericParam(eval.H{"ID": 99}, ""),
			expectedQuery: "DEFAULT CONTENT WHERE ID = ?",
			expectedArgs:  []any{99},
		},
		{
			name: "NodeGroupReturnsError",
			nodes: NodeGroup{
				&mockErrorNode{},
			},
			params:      emptyParams,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := OtherwiseNode{Nodes: tt.nodes}
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
