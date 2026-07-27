package xml

import (
	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
)

// group is the XML backend's node container. Bind declarations are scoped to
// the element in which they appear; they are deliberately not node.Node values.
type group struct {
	nodes node.Group
	binds bindNodeGroup
}

func (g group) Accept(translator driver.Translator, p eval.Parameter) (string, []any, error) {
	p = g.binds.ConvertParameter(p)
	return g.nodes.Accept(translator, p)
}

var _ node.Node = (*group)(nil)
