/*
Copyright 2023 eatmoreapple

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
	"errors"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
)

var ErrNilExpression = errors.New("juice: nil expression")

// ConditionNode is the shared implementation behind XML <if> and <when>
// elements. It renders its child nodes only when its expression evaluates to
// a non-zero value.
//
// It is not an XML element in its own right: IfNode and WhenNode use this
// implementation because they have the same condition-evaluation behavior.
type ConditionNode struct {
	expr      eval.Expression
	Nodes     node.Group
	BindNodes bindNodeGroup
}

// Parse compiles the XML test attribute into an evaluable expression.
// For example: "ID != nil", "age >= 18", or `status == "ACTIVE"`.
func (c *ConditionNode) Parse(test string) (err error) {
	c.expr, err = eval.Compile(test)
	return err
}

// Accept renders the child nodes when the condition matches.
func (c *ConditionNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	p = c.BindNodes.ConvertParameter(p)

	matched, err := c.Match(p)
	if err != nil {
		return "", nil, err
	}
	if !matched {
		return "", nil, nil
	}

	return c.Nodes.Accept(translator, p)
}

// Match evaluates the compiled expression against p. A non-zero result matches.
func (c *ConditionNode) Match(p eval.Parameter) (bool, error) {
	if c.expr == nil {
		return false, ErrNilExpression
	}

	value, err := c.expr.Execute(p)
	if err != nil {
		return false, err
	}
	return !value.IsZero(), nil
}

var _ node.Node = (*ConditionNode)(nil)
