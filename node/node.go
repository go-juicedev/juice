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

// Node is the syntax-independent runtime abstraction for a SQL fragment.
//
// Node deliberately defines no parsing or source-language semantics. An XML
// backend, a JSON backend, or a Go DSL may each provide different concrete
// implementations; the runtime only asks an implementation to render itself
// with a dialect translator and a parameter set.
//
// Accept returns the rendered SQL fragment, the arguments for its placeholders,
// and any evaluation error.
type Node interface {
	// Accept renders the node with the given translator and parameters.
	Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error)
}

// Group composes Nodes in source order into one syntax-independent Node.
type Group []Node

// Accept processes all Nodes in the group and combines their results.
// The method ensures proper spacing between node outputs and trims any extra whitespace.
// If the group is empty or no Nodes produce output, it returns empty results.
func (g Group) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	// Return early if group is empty
	nodeLength := len(g)
	switch nodeLength {
	case 0:
		return "", nil, nil
	case 1:
		return g[0].Accept(translator, p)
	}

	var builder strings.Builder

	// Pre-allocate string builder capacity to minimize buffer reallocations
	estimatedCapacity := nodeLength*12 + nodeLength - 1
	builder.Grow(estimatedCapacity)

	// Pre-allocate args slice to avoid reallocations
	args = make([]any, 0, nodeLength)

	lastIdx := nodeLength - 1

	// Process each node in the group
	for i, node := range g {
		q, a, err := node.Accept(translator, p)
		if err != nil {
			return "", nil, err
		}
		builder.WriteString(q)
		if len(a) > 0 {
			args = append(args, a...)
		}

		// Add space between Nodes, but only if something was written
		// and it's not the last node and doesn't already end with space.
		// Check q directly instead of builder.String() to avoid string allocation.
		if i < lastIdx && len(q) > 0 && q[len(q)-1] != ' ' {
			builder.WriteByte(' ')
		}
	}

	// Return empty results if no content was generated
	if builder.Len() == 0 {
		return "", nil, nil
	}

	return builder.String(), args, nil
}

var _ Node = (Group)(nil)
