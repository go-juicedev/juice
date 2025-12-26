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

package node

import (
	"errors"
	"strings"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

var ErrNilExpression = errors.New("juice: nil expression")

// ConditionNode represents a conditional SQL fragment with its evaluation expression and child Nodes.
// It is used to conditionally include or exclude SQL fragments based on runtime parameters.
type ConditionNode struct {
	expr      eval.Expression
	Nodes     Group
	BindNodes BindNodeGroup
}

// Parse compiles the given expression string into an evaluable expression.
// The expression syntax supports various operations like:
//   - Comparison: ==, !=, >, <, >=, <=
//   - Logical: &&, ||, !
//   - Null checks: != null, == null
//   - Property access: user.age, order.status
//
// Examples:
//
//	"ID != nil"              // Check for non-null
//	"age >= 18"               // Numeric comparison
//	"status == "ACTIVE""      // String comparison
//	"user.role == "ADMIN""    // Property access
func (c *ConditionNode) Parse(test string) (err error) {
	c.expr, err = eval.Compile(test)
	return err
}

func (c *ConditionNode) AcceptTo(translator driver.Translator, p eval.Parameter, builder *strings.Builder, args *[]any) error {
	p = c.BindNodes.ConvertParameter(p)

	matched, err := c.Match(p)
	if err != nil {
		return err
	}
	if !matched {
		return nil
	}

	return c.Nodes.AcceptTo(translator, p, builder, args)
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
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

// Match evaluates if the condition is true based on the provided parameter.
// It handles different types of values and converts them to boolean results:
//   - Bool: returns the boolean value directly
//   - Integers (signed/unsigned): returns true if non-zero
//   - Floats: returns true if non-zero
//   - String: returns true if non-empty
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

var _ NodeWriter = (*ConditionNode)(nil)
