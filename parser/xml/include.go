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
	"maps"
	"slices"
	"strings"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
)

type sqlRegistry struct {
	nodes        map[string]node.Node
	pending      map[string][]*IncludeNode
	dependencies map[string][]string
}

func (r *sqlRegistry) register(id string, n node.Node) error {
	if r.nodes == nil {
		r.nodes = make(map[string]node.Node)
	}
	if _, exists := r.nodes[id]; exists {
		return fmt.Errorf("duplicate SQL node %q", id)
	}
	r.nodes[id] = n
	for _, include := range r.pending[id] {
		include.sqlNode = n
	}
	delete(r.pending, id)
	return nil
}

func (r *sqlRegistry) bind(owner, id string, include *IncludeNode) {
	if owner != "" {
		if r.dependencies == nil {
			r.dependencies = make(map[string][]string)
		}
		r.dependencies[owner] = append(r.dependencies[owner], id)
	}
	if n, exists := r.nodes[id]; exists {
		include.sqlNode = n
		return
	}
	if r.pending == nil {
		r.pending = make(map[string][]*IncludeNode)
	}
	r.pending[id] = append(r.pending[id], include)
}

func (r *sqlRegistry) seal() error {
	if len(r.pending) > 0 {
		ids := slices.Sorted(maps.Keys(r.pending))
		return fmt.Errorf("SQL node %q not found", ids[0])
	}
	if cycle := r.findCycle(); len(cycle) > 0 {
		return fmt.Errorf("cyclic SQL include: %s", strings.Join(cycle, " -> "))
	}
	return nil
}

func (r *sqlRegistry) findCycle() []string {
	const (
		visiting = iota + 1
		visited
	)
	states := make(map[string]int, len(r.dependencies))
	stack := make([]string, 0, len(r.dependencies))

	var visit func(string) []string
	visit = func(id string) []string {
		switch states[id] {
		case visiting:
			start := slices.Index(stack, id)
			return append(slices.Clone(stack[start:]), id)
		case visited:
			return nil
		}

		states[id] = visiting
		stack = append(stack, id)
		dependencies := slices.Clone(r.dependencies[id])
		slices.Sort(dependencies)
		for _, dependency := range dependencies {
			if cycle := visit(dependency); len(cycle) > 0 {
				return cycle
			}
		}
		stack = stack[:len(stack)-1]
		states[id] = visited
		return nil
	}

	for _, id := range slices.Sorted(maps.Keys(r.dependencies)) {
		if cycle := visit(id); len(cycle) > 0 {
			return cycle
		}
	}
	return nil
}

type includeResolver struct {
	namespace string
	owner     string
	registry  *sqlRegistry
}

func (r *includeResolver) bind(refID string, include *IncludeNode) {
	id := refID
	if !strings.ContainsRune(id, '.') {
		id = r.namespace + "." + id
	}
	r.registry.bind(r.owner, id, include)
}

// IncludeNode represents a reference to another SQL fragment, enabling SQL reuse.
// It allows common SQL fragments to be defined once and included in multiple places,
// promoting code reuse and maintainability.
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
// The referenced SQL fragment is linked after all mapper sources have been
// parsed. References may point forward or across mapper namespaces, but every
// reference must be resolved before parsing completes. Cyclic fragment
// dependencies are rejected during the same linking phase.
type IncludeNode struct {
	sqlNode    node.Node
	properties eval.Parameter
}

// Accept accepts parameters and returns query and arguments.
func (i *IncludeNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
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
	include := &IncludeNode{}
	resolver.bind(refID, include)
	return include
}
