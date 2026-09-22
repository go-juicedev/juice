package xml

import (
	"errors"
	"reflect"
	"testing"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
)

type Node = node.Node
type Group = node.Group

var errMock = errors.New("mock error")

type mockErrorNode struct{}

func (*mockErrorNode) Accept(driver.Translator, eval.Parameter) (string, []any, error) {
	return "", nil, errMock
}

func equalArgs(a, b []any) bool { return reflect.DeepEqual(a, b) }

func newTestIf(t *testing.T, test string, nodes ...node.Node) *IfNode {
	t.Helper()
	compiled, err := NewIfNode(test)
	if err != nil {
		t.Fatalf("NewIfNode(%q) error = %v", test, err)
	}
	compiled.Nodes = nodes
	return compiled
}

func newTestWhen(t *testing.T, test string, nodes ...node.Node) *WhenNode {
	t.Helper()
	compiled, err := NewWhenNode(test)
	if err != nil {
		t.Fatalf("NewWhenNode(%q) error = %v", test, err)
	}
	compiled.Nodes = nodes
	return compiled
}
