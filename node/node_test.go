/*
Copyright 2026 eatmoreapple

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

type staticNode struct {
	query string
}

func (n staticNode) Accept(driver.Translator, eval.Parameter) (string, []any, error) {
	return n.query, nil, nil
}

func newStaticNode(query string) staticNode {
	return staticNode{
		query: query,
	}
}

func TestGroupJoinsNonEmptyNodes(t *testing.T) {
	tests := []struct {
		name  string
		nodes Group
		query string
	}{
		{
			name: "trailing empty node",
			nodes: Group{
				newStaticNode("name = ?,"),
				newStaticNode(""),
			},
			query: "name = ?,",
		},
		{
			name: "empty node between fragments",
			nodes: Group{
				newStaticNode("ID = ?"),
				newStaticNode(""),
				newStaticNode("AND name = ?"),
			},
			query: "ID = ? AND name = ?",
		},
		{
			name: "leading empty node",
			nodes: Group{
				newStaticNode(""),
				newStaticNode("ID = ?"),
			},
			query: "ID = ?",
		},
		{
			name: "previous fragment provides separator",
			nodes: Group{
				newStaticNode("SELECT "),
				newStaticNode("* FROM users"),
			},
			query: "SELECT * FROM users",
		},
		{
			name: "next fragment provides separator",
			nodes: Group{
				newStaticNode("SELECT"),
				newStaticNode(" * FROM users"),
			},
			query: "SELECT * FROM users",
		},
		{
			name: "all nodes empty",
			nodes: Group{
				newStaticNode(""),
				newStaticNode(""),
			},
			query: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := tt.nodes.Accept(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if query != tt.query {
				t.Fatalf("query = %q, want %q", query, tt.query)
			}
			if len(args) != 0 {
				t.Fatalf("args = %v, want none", args)
			}
		})
	}
}
