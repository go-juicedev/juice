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
	"fmt"
	"testing"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

type mockNodeManager struct {
	nodes map[string]Node
	err   error
	calls int
}

func (m *mockNodeManager) GetSQLNodeByID(id string) (Node, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	node, ok := m.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", id)
	}
	return node, nil
}

func TestIncludeNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()
	params := eval.NewGenericParam(eval.H{"ID": 1}, "")

	t.Run("PreLoadedNode", func(t *testing.T) {
		innerNode := NewTextNode("SELECT * FROM table WHERE ID = #{ID}")
		manager := &mockNodeManager{}
		node := NewIncludeNode(innerNode, manager, "ref")

		query, args, err := node.Accept(translator, params)
		if err != nil {
			t.Fatalf("Accept() error = %v", err)
		}
		if query != "SELECT * FROM table WHERE ID = ?" {
			t.Errorf("query = %s", query)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("args = %v", args)
		}
		if manager.calls != 0 {
			t.Errorf("manager calls = %d, want 0", manager.calls)
		}
	})

	t.Run("LazyLoadingSuccess", func(t *testing.T) {
		innerNode := NewTextNode("SELECT * FROM table WHERE ID = #{ID}")
		manager := &mockNodeManager{
			nodes: map[string]Node{
				"ref1": innerNode,
			},
		}
		node := NewIncludeNode(nil, manager, "ref1")

		query, args, err := node.Accept(translator, params)
		if err != nil {
			t.Fatalf("Accept() error = %v", err)
		}
		if query != "SELECT * FROM table WHERE ID = ?" {
			t.Errorf("query = %s", query)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("args = %v", args)
		}
		if manager.calls != 1 {
			t.Errorf("manager calls = %d, want 1", manager.calls)
		}

		// Second call should use cached node
		_, _, _ = node.Accept(translator, params)
		if manager.calls != 1 {
			t.Errorf("manager calls = %d, want 1 after second call", manager.calls)
		}
	})

	t.Run("LazyLoadingError", func(t *testing.T) {
		mockErr := errors.New("manager error")
		manager := &mockNodeManager{
			err: mockErr,
		}
		node := NewIncludeNode(nil, manager, "ref_err")

		_, _, err := node.Accept(translator, params)
		if !errors.Is(err, mockErr) {
			t.Errorf("err = %v, want %v", err, mockErr)
		}
		if manager.calls != 1 {
			t.Errorf("manager calls = %d, want 1", manager.calls)
		}
	})

	t.Run("LazyLoadingNotFound", func(t *testing.T) {
		manager := &mockNodeManager{
			nodes: make(map[string]Node),
		}
		node := NewIncludeNode(nil, manager, "missing")

		_, _, err := node.Accept(translator, params)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if manager.calls != 1 {
			t.Errorf("manager calls = %d, want 1", manager.calls)
		}
	})
}
