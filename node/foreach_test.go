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

func TestForeachNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	textNode := NewTextNode("(#{item.ID}, #{item.name})")
	node := ForeachNode{
		Nodes:      []Node{textNode},
		Item:       "item",
		Collection: "list",
		Separator:  ", ",
	}
	params := eval.H{"list": []map[string]any{
		{"ID": 1, "name": "a"},
		{"ID": 2, "name": "b"},
	}}
	query, args, err := node.Accept(drv.Translator(), params)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "(?, ?), (?, ?)" {
		t.Error("query error")
		return
	}
	if len(args) != 4 {
		t.Error("args error")
		return
	}
	if args[0] != 1 || args[1] != "a" || args[2] != 2 || args[3] != "b" {
		t.Error("args error")
		return
	}
}

func TestForeachNode_Accept_Comprehensive(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()

	type testCase struct {
		name           string
		collectionName string
		collectionData any
		itemVar        string
		indexVar       string
		open           string
		close          string
		separator      string
		nodes          []Node
		initialParams  eval.H
		expectedQuery  string
		expectedArgs   []any
		expectError    bool
		expectedErrMsg string
	}

	tests := []testCase{
		{
			name:           "EmptySliceCollection",
			collectionName: "list",
			collectionData: []any{},
			itemVar:        "val",
			nodes:          []Node{NewTextNode("#{val}")},
			expectedQuery:  "",
			expectedArgs:   nil,
		},
		{
			name:           "EmptyMapCollection",
			collectionName: "mapData",
			collectionData: map[string]any{},
			itemVar:        "val",
			indexVar:       "key",
			nodes:          []Node{NewTextNode("#{val}")},
			expectedQuery:  "",
			expectedArgs:   nil,
		},
		{
			name:           "SliceWithSimpleItems",
			collectionName: "ids",
			collectionData: []int{1, 2, 3},
			itemVar:        "ID",
			open:           "(",
			close:          ")",
			separator:      ",",
			nodes:          []Node{NewTextNode("#{ID}")},
			expectedQuery:  "(?,?,?)",
			expectedArgs:   []any{1, 2, 3},
		},
		{
			name:           "SliceWithStructsAndIndex",
			collectionName: "users",
			collectionData: []map[string]any{{"ID": 10, "name": "A"}, {"ID": 20, "name": "B"}},
			itemVar:        "user",
			indexVar:       "idx",
			open:           "VALUES ",
			separator:      ", ",
			nodes:          []Node{NewTextNode("(#{user.ID}, #{user.name}, index:#{idx})")},
			expectedQuery:  "VALUES (?, ?, index:?), (?, ?, index:?)",
			expectedArgs:   []any{10, "A", 0, 20, "B", 1},
		},
		{
			name:           "MapWithStringKeys",
			collectionName: "settings",
			collectionData: map[string]string{"host": "localhost", "port": "8080"},
			itemVar:        "settingsValue",
			indexVar:       "settingsKey",
			separator:      " AND ",
			nodes:          []Node{NewTextNode("#{settingsKey}=#{settingsValue}")},
			expectedQuery:  "?=? AND ?=?",
			expectedArgs:   []any{"host", "localhost", "port", "8080"},
		},
		{
			name:           "MapWithIntKeys",
			collectionName: "scores",
			collectionData: map[int]int{1: 100, 2: 200},
			itemVar:        "score",
			indexVar:       "playerId",
			separator:      "; ",
			nodes:          []Node{NewTextNode("Player #{playerId} scored #{score}")},
			expectedQuery:  "Player ? scored ?; Player ? scored ?",
			expectedArgs:   []any{1, 100, 2, 200},
		},
		{
			name:           "ItemNameConflict",
			collectionName: "data",
			collectionData: []int{1},
			itemVar:        "itemToConflict",
			nodes:          []Node{NewTextNode("#{itemToConflict}")},
			initialParams:  eval.H{"itemToConflict": "conflict"},
			expectError:    true,
			expectedErrMsg: "item itemToConflict already exists",
		},
		{
			name:           "CollectionNotFound",
			collectionName: "nonexistent",
			collectionData: nil,
			itemVar:        "val",
			nodes:          []Node{NewTextNode("#{val}")},
			expectError:    true,
			expectedErrMsg: "collection nonexistent not found",
		},
		{
			name:           "CollectionNotIterable_Int",
			collectionName: "num",
			collectionData: 123,
			itemVar:        "val",
			nodes:          []Node{NewTextNode("#{val}")},
			expectError:    true,
			expectedErrMsg: "collection num is not a slice or map",
		},
		{
			name:           "CollectionNotIterable_Struct",
			collectionName: "obj",
			collectionData: struct{ Name string }{"Test"},
			itemVar:        "val",
			nodes:          []Node{NewTextNode("#{val}")},
			expectError:    true,
			expectedErrMsg: "collection obj is not a slice or map",
		},
		{
			name:           "NodeReturnsErrorInLoop",
			collectionName: "items",
			collectionData: []string{"a", "b"},
			itemVar:        "i",
			nodes:          []Node{&mockErrorNode{}},
			expectError:    true,
			expectedErrMsg: "mock error",
		},
		{
			name:           "NoSeparator",
			collectionName: "parts",
			collectionData: []string{"one", "two"},
			itemVar:        "part",
			open:           "START:",
			close:          ":END",
			nodes:          []Node{NewTextNode("#{part}")},
			expectedQuery:  "START:??:END",
			expectedArgs:   []any{"one", "two"},
		},
		{
			name:           "OnlyOpen",
			collectionName: "elements",
			collectionData: []int{7, 8},
			itemVar:        "el",
			open:           "List: ",
			separator:      ",",
			nodes:          []Node{NewTextNode("#{el}")},
			expectedQuery:  "List: ?,?",
			expectedArgs:   []any{7, 8},
		},
		{
			name:           "OnlyClose",
			collectionName: "values",
			collectionData: []bool{true, false},
			itemVar:        "v",
			close:          ".",
			separator:      "|",
			nodes:          []Node{NewTextNode("#{v}")},
			expectedQuery:  "?|?.",
			expectedArgs:   []any{true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := eval.H{}
			if tt.initialParams != nil {
				for k, v := range tt.initialParams {
					params[k] = v
				}
			}
			if tt.expectedErrMsg != "collection "+tt.collectionName+" not found" {
				params[tt.collectionName] = tt.collectionData
			}

			node := ForeachNode{
				Collection: tt.collectionName,
				Nodes:      tt.nodes,
				Item:       tt.itemVar,
				Index:      tt.indexVar,
				Open:       tt.open,
				Close:      tt.close,
				Separator:  tt.separator,
			}

			query, args, err := node.Accept(translator, params)

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
				t.Errorf("Expected no error, but got %v", err)
				return
			}

			if tt.name == "MapWithStringKeys" || tt.name == "MapWithIntKeys" {
				if len(args) != len(tt.expectedArgs) {
					t.Errorf("Expected %d args, but got %d. Args: %v", len(tt.expectedArgs), len(args), args)
				}
				if query != tt.expectedQuery {
					if tt.name == "MapWithStringKeys" && query == "?=? AND ?=?" {
						altArgs := []any{"port", "8080", "host", "localhost"}
						if !equalArgs(args, altArgs) && !equalArgs(args, tt.expectedArgs) {
							t.Errorf("Expected query '%s' with args %v or %v, but got query '%s' with args %v", tt.expectedQuery, tt.expectedArgs, altArgs, query, args)
						}
					} else if tt.name == "MapWithIntKeys" && query == "Player ? scored ?; Player ? scored ?" {
						altArgs := []any{2, 200, 1, 100}
						if !equalArgs(args, altArgs) && !equalArgs(args, tt.expectedArgs) {
							t.Errorf("Expected query '%s' with args %v or %v, but got query '%s' with args %v", tt.expectedQuery, tt.expectedArgs, altArgs, query, args)
						}
					} else {
						t.Errorf("Expected query '%s', but got '%s' for map test", tt.expectedQuery, query)
					}
				}
			} else {
				if query != tt.expectedQuery {
					t.Errorf("Expected query '%s', but got '%s'", tt.expectedQuery, query)
				}
				if !equalArgs(args, tt.expectedArgs) {
					t.Errorf("Expected args %v, but got %v", tt.expectedArgs, args)
				}
			}
		})
	}
}

func TestForeachMapNode_Accept(t *testing.T) {
	drv := driver.MySQLDriver{}
	textNode := NewTextNode("(#{item}, #{index})")
	node := ForeachNode{
		Nodes:      []Node{textNode},
		Item:       "item",
		Index:      "index",
		Collection: "map",
		Separator:  ", ",
	}
	params := eval.H{"map": map[string]any{"a": 1}}
	query, args, err := node.Accept(drv.Translator(), params)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "(?, ?)" {
		t.Error("query error")
		return
	}
	if len(args) != 2 {
		t.Error("args error")
		return
	}
	if (args[0] != 1 || args[1] != "a") && (args[0] != "a" || args[1] != 1) {
		if args[0] != 1 || args[1] != "a" {
			t.Errorf("args error, got %v, %v, expected 1, a (or a, 1 for item, index)", args[0], args[1])
		}
	}

	params = eval.H{"map": map[string]any{"a": 1, "b": 2}}
	query, args, err = node.Accept(drv.Translator(), params)
	if err != nil {
		t.Error(err)
		return
	}
	if query != "(?, ?), (?, ?)" {
		t.Error("query error")
		return
	}
	if len(args) != 4 {
		t.Error("args error")
		return
	}
	expectedArgs1 := []any{1, "a", 2, "b"}
	expectedArgs2 := []any{2, "b", 1, "a"}
	if !equalArgs(args, expectedArgs1) && !equalArgs(args, expectedArgs2) {
		t.Errorf("Args error. Got %v. Expected %v or %v", args, expectedArgs1, expectedArgs2)
	}
}
