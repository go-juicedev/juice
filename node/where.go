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

// WhereNode represents a SQL WHERE clause and its conditions.
// It manages a group of condition Nodes that form the complete WHERE clause.
type WhereNode struct {
	Nodes     NodeGroup
	BindNodes BindNodeGroup
}

// Accept processes the WHERE clause and its conditions.
// It handles several special cases:
//  1. Removes leading "AND" or "OR" from the first condition
//  2. Ensures the clause starts with "WHERE" if not already present
//  3. Properly handles spacing between conditions
//
// Examples:
//
//	Input:  "AND ID = ?"        -> Output: "WHERE ID = ?"
//	Input:  "OR name = ?"       -> Output: "WHERE name = ?"
//	Input:  "WHERE age > ?"     -> Output: "WHERE age > ?"
//	Input:  "status = ?"        -> Output: "WHERE status = ?"
func (w WhereNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	p = w.BindNodes.ConvertParameter(p)

	query, args, err = w.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}

	if query == "" {
		return "", args, nil
	}
	// A space is required at the end; otherwise, it is meaningless.
	switch {
	case strings.HasPrefix(query, "and ") || strings.HasPrefix(query, "AND "):
		query = query[4:]
	case strings.HasPrefix(query, "or ") || strings.HasPrefix(query, "OR "):
		query = query[3:]
	}

	// A space is required at the end; otherwise, it is meaningless.
	if !strings.HasPrefix(query, "where ") && !strings.HasPrefix(query, "WHERE ") {
		query = "WHERE " + query
	}
	return
}

var _ Node = (*WhereNode)(nil)
