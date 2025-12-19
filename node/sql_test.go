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

func TestSQLNode_AcceptAndID(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()

	t.Run("IDMethod", func(t *testing.T) {
		expectedID := "testSQLNodeID"
		node := SQLNode{ID: expectedID}
		if node.ID != expectedID {
			t.Errorf("Expected ID '%s', but got '%s'", expectedID, node.ID)
		}
	})

	t.Run("Accept_EmptyNodes", func(t *testing.T) {
		sqlNode := SQLNode{ID: "empty", Nodes: NodeGroup{}}
		query, args, err := sqlNode.Accept(translator, eval.NewGenericParam(eval.H{}, ""))
		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
		if query != "" {
			t.Errorf("Expected empty query, but got '%s'", query)
		}
		if len(args) != 0 {
			t.Errorf("Expected no args, but got %v", args)
		}
	})

	t.Run("Accept_WithNodes", func(t *testing.T) {
		nodes := NodeGroup{
			NewTextNode("SELECT * FROM table WHERE ID = #{ID}"),
		}
		sqlNode := SQLNode{ID: "selectUser", Nodes: nodes}
		params := eval.NewGenericParam(eval.H{"ID": 123}, "")
		query, args, err := sqlNode.Accept(translator, params)

		if err != nil {
			t.Errorf("Expected no error, but got %v", err)
		}
		expectedQuery := "SELECT * FROM table WHERE ID = ?"
		if query != expectedQuery {
			t.Errorf("Expected query '%s', but got '%s'", expectedQuery, query)
		}
		expectedArgs := []any{123}
		if !equalArgs(args, expectedArgs) {
			t.Errorf("Expected args %v, but got %v", expectedArgs, args)
		}
	})

	t.Run("Accept_NodeReturnsError", func(t *testing.T) {
		nodes := NodeGroup{
			&mockErrorNode{},
		}
		sqlNode := SQLNode{ID: "errorNode", Nodes: nodes}
		_, _, err := sqlNode.Accept(translator, eval.NewGenericParam(eval.H{}, ""))
		if err == nil {
			t.Errorf("Expected an error, but got nil")
		}
		if err != errMock {
			t.Errorf("Expected errMock, but got %v", err)
		}
	})
}
