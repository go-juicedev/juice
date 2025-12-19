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
	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

// ChooseNode implements a switch-like conditional structure for SQL generation.
// It evaluates multiple conditions in order and executes the first matching case,
// with an optional default case (otherwise).
//
// Fields:
//   - WhenNodes: Ordered list of conditional branches to evaluate
//   - OtherwiseNode: Default branch if no when conditions match
//
// Example XML:
//
//	<choose>
//	  <when test="ID != 0">
//	    AND ID = #{ID}
//	  </when>
//	  <when test='name != ""'>
//	    AND name LIKE CONCAT('%', #{name}, '%')
//	  </when>
//	  <otherwise>
//	    AND status = 'ACTIVE'
//	  </otherwise>
//	</choose>
//
// Behavior:
//  1. Evaluates each <when> condition in order
//  2. Executes SQL from first matching condition
//  3. If no conditions match, executes <otherwise> if present
//  4. If no conditions match and no otherwise, returns empty result
//
// Usage scenarios:
//  1. Complex conditional logic in WHERE clauses
//  2. Dynamic sorting options
//  3. Different JOIN conditions
//  4. Status-based queries
//
// Example results:
//
//	Case 1 (ID present):
//	  AND ID = ?
//	Case 2 (only name present):
//	  AND name LIKE ?
//	Case 3 (neither present):
//	  AND status = 'ACTIVE'
//
// Note: Similar to a switch statement in programming languages,
// only the first matching condition is executed.
type ChooseNode struct {
	WhenNodes     []Node
	OtherwiseNode Node
	BindNodes     BindNodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (c ChooseNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	p = c.BindNodes.ConvertParameter(p)

	for _, node := range c.WhenNodes {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		// if one of when Nodes is true, return query and arguments
		if len(q) > 0 {
			return q, a, nil
		}
	}

	// if all when Nodes are false, return otherwise node
	if c.OtherwiseNode != nil {
		return c.OtherwiseNode.Accept(translator, p)
	}
	return "", nil, nil
}

var _ Node = (*ChooseNode)(nil)
