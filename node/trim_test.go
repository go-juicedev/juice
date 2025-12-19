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

func TestTrimNode_Accept_Comprehensive(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()
	emptyParams := eval.NewGenericParam(eval.H{}, "")

	tests := []struct {
		name            string
		nodes           NodeGroup
		prefix          string
		prefixOverrides []string
		suffix          string
		suffixOverrides []string
		params          eval.Parameter
		expectedQuery   string
		expectedArgs    []any
		expectError     bool
	}{
		{
			name:          "AllEmpty",
			nodes:         NodeGroup{NewTextNode("content")},
			params:        emptyParams,
			expectedQuery: "content",
		},
		{
			name:          "OnlyPrefix",
			nodes:         NodeGroup{NewTextNode("content")},
			prefix:        "PRE-",
			params:        emptyParams,
			expectedQuery: "PRE-content",
		},
		{
			name:          "OnlySuffix",
			nodes:         NodeGroup{NewTextNode("content")},
			suffix:        "-SUF",
			params:        emptyParams,
			expectedQuery: "content-SUF",
		},
		{
			name:            "PrefixOverrideMatch",
			nodes:           NodeGroup{NewTextNode("OVERRIDE_ME content")},
			prefix:          "NEW_PRE-",
			prefixOverrides: []string{"OVERRIDE_ME ", "OTHER_"},
			params:          emptyParams,
			expectedQuery:   "NEW_PRE-content",
		},
		{
			name:            "PrefixOverrideNoMatch",
			nodes:           NodeGroup{NewTextNode("NO_MATCH content")},
			prefix:          "PRE-",
			prefixOverrides: []string{"OVERRIDE_ME "},
			params:          emptyParams,
			expectedQuery:   "PRE-NO_MATCH content",
		},
		{
			name:            "SuffixOverrideMatch",
			nodes:           NodeGroup{NewTextNode("content SUFF_OVERRIDE")},
			suffix:          "-NEW_SUF",
			suffixOverrides: []string{" SUFF_OVERRIDE", " _OTHER"},
			params:          emptyParams,
			expectedQuery:   "content-NEW_SUF",
		},
		{
			name:            "SuffixOverrideNoMatch",
			nodes:           NodeGroup{NewTextNode("content NO_MATCH")},
			suffix:          "-SUF",
			suffixOverrides: []string{" SUFF_OVERRIDE"},
			params:          emptyParams,
			expectedQuery:   "content NO_MATCH-SUF",
		},
		{
			name:            "AllAttributesSet_OverridesMatch",
			nodes:           NodeGroup{NewTextNode("PRE_OV content SUF_OV")},
			prefix:          "PREFIX ",
			prefixOverrides: []string{"PRE_OV "},
			suffix:          " SUFFIX",
			suffixOverrides: []string{" SUF_OV"},
			params:          emptyParams,
			expectedQuery:   "PREFIX content SUFFIX",
		},
		{
			name:            "AllAttributesSet_OverridesNoMatch",
			nodes:           NodeGroup{NewTextNode("original content")},
			prefix:          "PREFIX ",
			prefixOverrides: []string{"NO_PRE_OV "},
			suffix:          " SUFFIX",
			suffixOverrides: []string{" NO_SUF_OV"},
			params:          emptyParams,
			expectedQuery:   "PREFIX original content SUFFIX",
		},
		{
			name:          "ChildNodesReturnEmptyQuery",
			nodes:         NodeGroup{&IfNode{}},
			prefix:        "PRE-",
			suffix:        "-SUF",
			params:        emptyParams,
			expectedQuery: "",
		},
		{
			name: "ChildNodeReturnsError",
			nodes: NodeGroup{
				&mockErrorNode{},
			},
			params:      emptyParams,
			expectError: true,
		},
		{
			name:            "PrefixOverrideWithSpaceAtEnd_ContentAlsoHasSpace",
			nodes:           NodeGroup{NewTextNode("AND ID = 1")},
			prefix:          "WHERE ",
			prefixOverrides: []string{"AND ", "OR "},
			params:          emptyParams,
			expectedQuery:   "WHERE ID = 1",
		},
		{
			name:            "SuffixOverrideWithSpaceAtStart_ContentAlsoHasSpace",
			nodes:           NodeGroup{NewTextNode("ID = 1 ,")},
			suffix:          ";",
			suffixOverrides: []string{" ,", " ;"},
			params:          emptyParams,
			expectedQuery:   "ID = 1;",
		},
		{
			name:            "EmptyPrefixOverrides",
			nodes:           NodeGroup{NewTextNode("AND content")},
			prefix:          "WHERE ",
			prefixOverrides: []string{},
			params:          emptyParams,
			expectedQuery:   "WHERE AND content",
		},
		{
			name:            "EmptySuffixOverrides",
			nodes:           NodeGroup{NewTextNode("content,")},
			suffix:          ";",
			suffixOverrides: []string{},
			params:          emptyParams,
			expectedQuery:   "content,;",
		},
		{
			name:          "ContentGeneratedWithArgs",
			nodes:         NodeGroup{NewTextNode("ID = #{ID}")},
			prefix:        "WHERE ",
			params:        eval.NewGenericParam(eval.H{"ID": 123}, ""),
			expectedQuery: "WHERE ID = ?",
			expectedArgs:  []any{123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, n := range tt.nodes {
				if ifNode, ok := n.(*IfNode); ok {
					if err := ifNode.Parse("false"); err != nil {
						t.Fatalf("Failed to parse IfNode condition for test %s: %v", tt.name, err)
					}
				}
			}

			node := TrimNode{
				Nodes:           tt.nodes,
				Prefix:          tt.prefix,
				PrefixOverrides: tt.prefixOverrides,
				Suffix:          tt.suffix,
				SuffixOverrides: tt.suffixOverrides,
			}
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

func TestTrimNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	node1 := NewTextNode("name,")
	ifNode := &IfNode{
		Nodes: []Node{node1},
	}
	if err := ifNode.Parse("ID > 0"); err != nil {
		t.Error(err)
		return
	}
	node := &TrimNode{
		Nodes: []Node{
			ifNode,
		},
		Prefix:          "(",
		Suffix:          ")",
		SuffixOverrides: []string{","},
	}
	params := eval.H{"ID": 1, "name": "a"}
	query, args, err := node.Accept(drv.Translator(), params)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "(name)" {
		t.Log(query)
		t.Error("query error")
		return
	}
	if len(args) != 0 {
		t.Error("args error")
		return
	}
}
