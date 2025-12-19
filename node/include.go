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

type nodeManager interface {
	GetSQLNodeByID(id string) (Node, error)
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
	sqlNode Node
	manager nodeManager
	refId   string
}

// Accept accepts parameters and returns query and arguments.
func (i *IncludeNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	if i.sqlNode == nil {
		// lazy loading
		// does it need to be thread safe?
		sqlNode, err := i.manager.GetSQLNodeByID(i.refId)
		if err != nil {
			return "", nil, err
		}
		i.sqlNode = sqlNode
	}

	return i.sqlNode.Accept(translator, p)
}

func NewIncludeNode(sqlNode Node, manager nodeManager, refId string) *IncludeNode {
	return &IncludeNode{
		sqlNode: sqlNode,
		manager: manager,
		refId:   refId,
	}
}
