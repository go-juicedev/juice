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
	"strings"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

// SetNode represents an SQL SET clause for UPDATE statements.
// It manages a group of assignment expressions and automatically handles
// the comma separators and SET prefix.
//
// Features:
//   - Automatically adds "SET" prefix
//   - Manages comma separators between assignments
//   - Handles dynamic assignments based on conditions
//
// Example XML:
//
//	<update ID="updateUser">
//	  UPDATE users
//	  <set>
//	    <if test='name != ""'>
//	      name = #{name},
//	    </if>
//	    <if test="age > 0">
//	      age = #{age},
//	    </if>
//	    <if test="status != 0">
//	      status = #{status}
//	    </if>
//	  </set>
//	  WHERE ID = #{ID}
//	</update>
//
// Example results:
//
//	Case 1 (name and age set):
//	  UPDATE users SET name = ?, age = ? WHERE ID = ?
//
//	Case 2 (only status set):
//	  UPDATE users SET status = ? WHERE ID = ?
//
// Note: The node automatically handles trailing commas and ensures
// proper formatting of the SET clause regardless of which fields
// are included dynamically.
type SetNode struct {
	Nodes     Group
	BindNodes BindNodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (s SetNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	p = s.BindNodes.ConvertParameter(p)

	query, args, err = s.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", args, nil
	}

	// Remove trailing comma
	query = strings.TrimSuffix(query, ",")

	// Ensure SET prefix if not present
	if !strings.HasPrefix(query, "set ") && !strings.HasPrefix(query, "SET ") {
		query = "SET " + query
	}

	return query, args, nil
}

var _ Node = (*SetNode)(nil)
