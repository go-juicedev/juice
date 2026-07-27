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

func parseExprNoError(t *testing.T, expression string) eval.Expression {
	t.Helper()
	compiled, err := eval.Compile(expression)
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

type testNodeResolver interface {
	GetSQLNodeByID(string) (node.Node, error)
}

type deferredTestNode struct {
	resolver testNodeResolver
	refID    string
	node     node.Node
}

func (n *deferredTestNode) Accept(translator driver.Translator, p eval.Parameter) (string, []any, error) {
	if n.node == nil {
		resolved, err := n.resolver.GetSQLNodeByID(n.refID)
		if err != nil {
			return "", nil, err
		}
		n.node = resolved
	}
	return n.node.Accept(translator, p)
}

func NewIncludeNode(sqlNode node.Node, resolver testNodeResolver, refID string) *IncludeNode {
	registry := &sqlRegistry{}
	linked := &includeResolver{namespace: "test", registry: registry}
	include := newIncludeNode(refID, linked)
	include.sqlNode = sqlNode
	if sqlNode == nil {
		_ = registry.register("test."+refID, &deferredTestNode{resolver: resolver, refID: refID})
	}
	return include
}
