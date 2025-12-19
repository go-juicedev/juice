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
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/internal/reflectlite"
)

var (
	// paramRegex matches parameter placeholders in SQL queries using #{...} syntax.
	// Examples:
	//   - #{ID}         -> matches
	//   - #{user.name}  -> matches
	//   - #{  age  }    -> matches (whitespace is ignored)
	//   - #{}           -> doesn't match (requires identifier)
	//   - #{123}        -> matches
	paramRegex = regexp.MustCompile(`#{\s*(\w+(?:\.\w+)*)\s*}`)

	// formatRegexp matches string interpolation placeholders using ${...} syntax.
	// Unlike paramRegex, these are replaced directly in the SQL string.
	// WARNING: Be careful with this as it can lead to SQL injection if not properly sanitized.
	// Examples:
	//   - ${tableName}  -> matches
	//   - ${db.schema}  -> matches
	//   - ${  field  }  -> matches (whitespace is ignored)
	//   - ${}           -> doesn't match (requires identifier)
	//   - ${123}        -> matches
	formatRegexp = regexp.MustCompile(`\${\s*(\w+(?:\.\w+)*)\s*}`)
)

// Node is the fundamental interface for all SQL generation components.
// It defines the contract for converting dynamic SQL structures into
// concrete SQL queries with their corresponding parameters.
//
// The Accept method follows the Visitor pattern, allowing different
// SQL dialects to be supported through the translator parameter.
//
// Parameters:
//   - translator: Handles dialect-specific SQL translations
//   - parameter: Contains parameter values for SQL generation
//
// Returns:
//   - query: The generated SQL fragment
//   - args: Slice of arguments for prepared statement
//   - err: Any error during SQL generation
//
// Implementing types include:
//   - SQLNode: Complete SQL statements
//   - WhereNode: WHERE clause handling
//   - SetNode: SET clause for updates
//   - IfNode: Conditional inclusion
//   - ChooseNode: Switch-like conditionals
//   - ForeachNode: Collection iteration
//   - TrimNode: String manipulation
//   - IncludeNode: SQL fragment reuse
//
// Example usage:
//
//	query, args, err := node.Accept(mysqlTranslator, params)
//	if err != nil {
//	  // handle error
//	}
//	// use query and args with database
//
// Note: Implementations should handle their specific SQL generation
// logic while maintaining consistency with the overall SQL structure.
type Node interface {
	// Accept processes the node with given translator and parameters
	// to produce a SQL fragment and its arguments.
	Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error)
}

// NodeGroup wraps multiple Nodes into a single node.
type NodeGroup []Node

// Accept processes all Nodes in the group and combines their results.
// The method ensures proper spacing between node outputs and trims any extra whitespace.
// If the group is empty or no Nodes produce output, it returns empty results.
func (g NodeGroup) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	// Return early if group is empty
	nodeLength := len(g)
	switch nodeLength {
	case 0:
		return "", nil, nil
	case 1:
		return g[0].Accept(translator, p)
	}

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

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
		if len(q) > 0 {
			builder.WriteString(q)

			// Add space between Nodes, but not after the last one
			if i < lastIdx && !strings.HasSuffix(q, " ") {
				builder.WriteString(" ")
			}
		}
		if len(a) > 0 {
			args = append(args, a...)
		}
	}

	// Return empty results if no content was generated
	if builder.Len() == 0 {
		return "", nil, nil
	}

	return builder.String(), args, nil
}

// reflectValueToString converts reflect.Value to string
func reflectValueToString(v reflect.Value) string {
	v = reflectlite.Unwrap(v)
	switch t := v.Interface().(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(v.Bytes())
	case fmt.Stringer:
		return t.String()
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(v.Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case float32:
		return strconv.FormatFloat(v.Float(), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	case bool:
		return strconv.FormatBool(v.Bool())
	default:
		return fmt.Sprintf("%v", t)
	}
}

// bindScope provides lookup and execution of bind variables within a scope.
type bindScope struct {
	nodes     []*BindNode
	parameter eval.Parameter
}

// Get finds a BindNode by name and executes it using the scope's parameter.
// Returns ErrBindVariableNotFound wrapped if no bind with the given name exists.
func (b bindScope) Get(name string) (reflect.Value, error) {
	for _, bind := range b.nodes {
		if bind.Name == name {
			return bind.Execute(b.parameter)
		}
	}
	return reflect.Value{}, fmt.Errorf("%w: %s", ErrBindVariableNotFound, name)
}
