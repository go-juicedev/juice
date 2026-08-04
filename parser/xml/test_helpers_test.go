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
