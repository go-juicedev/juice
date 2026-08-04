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

package xml

import (
	"strings"
	"testing"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

func TestSQLRegistryLinksIncludes(t *testing.T) {
	registry := &sqlRegistry{}
	local := NewTextNode("local")
	remote := NewTextNode("remote")
	resolver := &includeResolver{namespace: "current.Mapper", registry: registry}

	localInclude := newIncludeNode("local", resolver)
	if err := registry.register("current.Mapper.local", local); err != nil {
		t.Fatal(err)
	}
	if err := registry.register("other.Mapper.remote", remote); err != nil {
		t.Fatal(err)
	}
	remoteInclude := newIncludeNode("other.Mapper.remote", resolver)

	if err := registry.register("other.Mapper.remote", remote); err == nil {
		t.Fatal("duplicate SQL node was accepted")
	}
	if err := registry.seal(); err != nil {
		t.Fatal(err)
	}
	if localInclude.sqlNode != local {
		t.Fatal("local include was not linked")
	}
	if remoteInclude.sqlNode != remote {
		t.Fatal("cross-namespace include was not linked")
	}
}

func TestSQLRegistrySealRejectsMissingInclude(t *testing.T) {
	registry := &sqlRegistry{}
	resolver := &includeResolver{namespace: "current.Mapper", registry: registry}
	newIncludeNode("missing", resolver)

	err := registry.seal()
	if err == nil || !strings.Contains(err.Error(), `SQL node "current.Mapper.missing" not found`) {
		t.Fatalf("seal() error = %v", err)
	}
}

func TestIncludeNode_Accept_include_test(t *testing.T) {
	drv := driver.MySQLDriver{}
	translator := drv.Translator()
	params := eval.NewGenericParam(eval.H{"ID": 1}, "")

	t.Run("PreLoadedNode", func(t *testing.T) {
		innerNode := NewTextNode("SELECT * FROM table WHERE ID = #{ID}")
		node := &IncludeNode{sqlNode: innerNode}

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
	})

	t.Run("PropertiesOverrideParentParameter", func(t *testing.T) {
		innerNode := NewTextNode("SELECT ${columns} FROM ${table} WHERE ID = #{ID}")
		node := (&IncludeNode{sqlNode: innerNode}).WithProperties(eval.H{
			"columns": "ID, name",
			"table":   "users",
		})

		query, args, err := node.Accept(
			translator,
			eval.NewGenericParam(eval.H{
				"ID":      1,
				"columns": "ignored",
				"table":   "ignored",
			}, ""),
		)
		if err != nil {
			t.Fatalf("Accept() error = %v", err)
		}
		if query != "SELECT ID, name FROM users WHERE ID = ?" {
			t.Errorf("query = %s", query)
		}
		if len(args) != 1 || args[0] != 1 {
			t.Errorf("args = %v", args)
		}
	})
}
