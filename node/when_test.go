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

func TestWhenNode_Accept_when_test(t *testing.T) {
	drv := driver.MySQLDriver{}
	node1 := NewTextNode("select * from user where ID = #{ID}")
	var node = WhenNode{
		Nodes: []Node{node1},
	}

	if err := node.Parse("ID > 0"); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	paramsTrue := eval.H{"ID": 1}
	query, args, err := node.Accept(drv.Translator(), paramsTrue)
	if err != nil {
		t.Errorf("Accept() error = %v for true condition", err)
	}
	if query != "select * from user where ID = ?" {
		t.Errorf("Accept() query = %s, want 'select * from user where ID = ?' for true condition", query)
	}
	if len(args) != 1 || args[0] != 1 {
		t.Errorf("Accept() args = %v, want [1] for true condition", args)
	}

	paramsFalse := eval.H{"ID": 0}
	query, args, err = node.Accept(drv.Translator(), paramsFalse)
	if err != nil {
		t.Errorf("Accept() error = %v for false condition", err)
	}
	if query != "" {
		t.Errorf("Accept() query = %s, want '' for false condition", query)
	}
	if len(args) != 0 {
		t.Errorf("Accept() args = %v, want [] for false condition", args)
	}
}
