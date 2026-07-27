package xml

import (
	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
)

// SQL represents an XML <sql> element referenced by <include>.
type SQL struct {
	id   string
	node node.Node
}

func (s *SQL) ID() string { return s.id }

func (s *SQL) Accept(translator driver.Translator, p eval.Parameter) (string, []any, error) {
	return s.node.Accept(translator, p)
}

var _ node.Node = (*SQL)(nil)
