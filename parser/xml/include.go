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
	"fmt"
	"strings"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
)

type sqlRegistry struct {
	nodes map[string]node.Node
}

func (r *sqlRegistry) register(id string, n node.Node) error {
	if r.nodes == nil {
		r.nodes = make(map[string]node.Node)
	}
	if _, exists := r.nodes[id]; exists {
		return fmt.Errorf("duplicate SQL node %q", id)
	}
	r.nodes[id] = n
	return nil
}

type includeResolver struct {
	namespace string
	registry  *sqlRegistry
}

func (r *includeResolver) resolve(refID string) (node.Node, error) {
	if r == nil || r.registry == nil {
		return nil, fmt.Errorf("XML include %q has no SQL registry", refID)
	}
	id := refID
	if !strings.ContainsRune(id, '.') {
		id = r.namespace + "." + id
	}
	n, exists := r.registry.nodes[id]
	if !exists {
		return nil, fmt.Errorf("SQL node %q not found", id)
	}
	return n, nil
}

// IncludeNode represents a reference to another SQL fragment, enabling SQL reuse.
// It allows common SQL fragments to be defined once and included in multiple places,
// promoting code reuse and maintainability.
//
// Fields:
//   - sqlNode: The referenced SQL fragment node
//   - mapper: Reference to the parent Mapper for context
//   - refId: ID of the SQL fragment to include
//
// Example XML:
//
//	<!-- Common WHERE clause -->
//	<sql ID="userFields">
//	  ID, name, age, status
//	</sql>
//
//	<!-- Using the include -->
//	<select ID="getUsers">
//	  SELECT
//	  <include refid="userFields"/>
//	  FROM users
//	  WHERE status = #{status}
//	</select>
//
// Features:
//   - Enables SQL fragment reuse
//   - Supports cross-mapper references
//   - Maintains consistent SQL patterns
//   - Reduces code duplication
//
// Usage scenarios:
//  1. Common column lists
//  2. Shared WHERE conditions
//  3. Reusable JOIN clauses
//  4. Standard filtering conditions
//
// Note: The refId must reference an existing SQL fragment defined with
// the <sql> tag. The reference can be within the same mapper or from
// another mapper if properly configured.
type IncludeNode struct {
	sqlNode    node.Node
	resolver   *includeResolver
	refId      string
	properties eval.Parameter
}

// Accept accepts parameters and returns query and arguments.
func (i *IncludeNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	if i.sqlNode == nil {
		sqlNode, err := i.resolver.resolve(i.refId)
		if err != nil {
			return "", nil, err
		}
		i.sqlNode = sqlNode
	}

	if i.properties != nil {
		p = eval.ParamGroup{i.properties, p}
	}

	return i.sqlNode.Accept(translator, p)
}

func (i *IncludeNode) WithProperties(properties eval.Parameter) *IncludeNode {
	i.properties = properties
	return i
}

func newIncludeNode(refID string, resolver *includeResolver) *IncludeNode {
	return &IncludeNode{
		refId:    refID,
		resolver: resolver,
	}
}
