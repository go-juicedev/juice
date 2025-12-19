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

// SQLNode represents a complete SQL statement with its metadata and child Nodes.
// It serves as the root node for a single SQL operation (SELECT, INSERT, UPDATE, DELETE)
// and manages the entire SQL generation process.
//
// Fields:
//   - ID: Unique identifier for the SQL statement within the mapper
//   - Nodes: Collection of child Nodes that form the complete SQL
//   - mapper: Reference to the parent Mapper for context and configuration
//
// Example XML:
//
//	<select ID="getUserById">
//	  SELECT *
//	  FROM users
//	  <where>
//	    <if test="ID != 0">
//	      ID = #{ID}
//	    </if>
//	  </where>
//	</select>
//
// Usage scenarios:
//  1. SELECT statements with dynamic conditions
//  2. INSERT statements with optional fields
//  3. UPDATE statements with dynamic SET clauses
//  4. DELETE statements with complex WHERE conditions
//
// Features:
//   - Manages complete SQL statement generation
//   - Handles parameter binding
//   - Supports dynamic SQL through child Nodes
//   - Maintains connection to mapper context
//   - Enables statement reuse through ID reference
//
// Note: The ID must be unique within its mapper context to allow
// proper statement lookup and execution.
type SQLNode struct {
	ID        string    // Unique identifier for the SQL statement
	Nodes     NodeGroup // Child Nodes forming the SQL statement
	BindNodes BindNodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (s SQLNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	p = s.BindNodes.ConvertParameter(p)

	return s.Nodes.Accept(translator, p)
}

var _ Node = (*SQLNode)(nil)
