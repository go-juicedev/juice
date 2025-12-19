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

// TrimNode handles SQL fragment cleanup by managing prefixes, suffixes, and their overrides.
// It's particularly useful for dynamically generated SQL where certain prefixes or suffixes
// might need to be added or removed based on the context.
//
// Fields:
//   - Nodes: Group of child Nodes containing the SQL fragments
//   - Prefix: String to prepend to the result if content exists
//   - PrefixOverrides: Strings to remove if found at the start
//   - Suffix: String to append to the result if content exists
//   - SuffixOverrides: Strings to remove if found at the end
//
// Common use cases:
//  1. Removing leading AND/OR from WHERE clauses
//  2. Managing commas in clauses
//  3. Handling dynamic UPDATE SET statements
//
// Example XML:
//
//	<trim prefix="WHERE" prefixOverrides="AND|OR">
//	  <if test="ID > 0">
//	    AND ID = #{ID}
//	  </if>
//	  <if test='name != ""'>
//	    AND name = #{name}
//	  </if>
//	</trim>
//
// Example Result:
//
//	Input:  "AND ID = ? AND name = ?"
//	Output: "WHERE ID = ? AND name = ?"
type TrimNode struct {
	Nodes           NodeGroup
	Prefix          string
	PrefixOverrides []string
	Suffix          string
	SuffixOverrides []string
	BindNodes       BindNodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (t TrimNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	p = t.BindNodes.ConvertParameter(p)

	query, args, err = t.Nodes.Accept(translator, p)
	if err != nil {
		return "", nil, err
	}

	if len(query) == 0 {
		return "", nil, nil
	}

	// Handle prefix overrides before adding prefix
	if len(t.PrefixOverrides) > 0 {
		for _, prefix := range t.PrefixOverrides {
			if strings.HasPrefix(query, prefix) {
				query = query[len(prefix):]
				break
			}
		}
	}

	// Handle suffix overrides before adding suffix
	if len(t.SuffixOverrides) > 0 {
		for _, suffix := range t.SuffixOverrides {
			if strings.HasSuffix(query, suffix) {
				query = query[:len(query)-len(suffix)]
				break
			}
		}
	}

	// Build final query with prefix and suffix
	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	builder.Grow(len(t.Prefix) + len(query) + len(t.Suffix))

	if t.Prefix != "" {
		builder.WriteString(t.Prefix)
	}
	builder.WriteString(query)
	if t.Suffix != "" {
		builder.WriteString(t.Suffix)
	}

	return builder.String(), args, nil
}

var _ Node = (*TrimNode)(nil)
