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

// OtherwiseNode represents the default branch in a <choose> statement,
// which executes when none of the <when> conditions are met.
// It's similar to the 'default' case in a switch statement.
//
// Fields:
//   - Nodes: Group of Nodes containing the default SQL fragments
//
// Example XML:
//
//	<choose>
//	  <when test="status != nil">
//	    AND status = #{status}
//	  </when>
//	  <when test="type != nil">
//	    AND type = #{type}
//	  </when>
//	  <otherwise>
//	    AND is_deleted = 0
//	    AND status = 'ACTIVE'
//	  </otherwise>
//	</choose>
//
// Behavior:
//   - Executes only if all <when> conditions are false
//   - No condition evaluation needed
//   - Can contain multiple SQL fragments
//   - Optional within <choose> block
//
// Usage scenarios:
//  1. Default filtering conditions
//  2. Fallback sorting options
//  3. Default join conditions
//  4. Error prevention (ensuring non-empty WHERE clauses)
//
// Example results:
//
//	When no conditions match:
//	  AND is_deleted = 0 AND status = 'ACTIVE'
//
// Note: Unlike WhenNode, OtherwiseNode doesn't evaluate any conditions.
// It simply provides default SQL fragments when needed.
type OtherwiseNode struct {
	Nodes     Group
	BindNodes BindNodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (o OtherwiseNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	p = o.BindNodes.ConvertParameter(p)

	return o.Nodes.Accept(translator, p)
}

var _ Node = (*OtherwiseNode)(nil)
