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

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

// ForeachNode represents a dynamic SQL fragment that iterates over a collection.
// It's commonly used for IN clauses, batch inserts, or any scenario requiring
// iteration over a collection of values in SQL generation.
//
// Fields:
//   - Collection: Expression to get the collection to iterate over
//   - Nodes: SQL fragments to be repeated for each item
//   - Item: Variable name for the current item in iteration
//   - Index: Variable name for the current index (optional)
//   - Open: String to prepend before the iteration results
//   - Close: String to append after the iteration results
//   - Separator: String to insert between iterations
//
// Example XML:
//
//	<foreach collection="list" item="item" index="i" open="(" separator="," close=")">
//	  #{item}
//	</foreach>
//
// Usage scenarios:
//
//  1. IN clauses:
//     WHERE ID IN (#{item})
//
//  2. Batch inserts:
//     INSERT INTO users VALUES
//     <foreach collection="users" item="user" separator=",">
//     (#{user.ID}, #{user.name})
//     </foreach>
//
//  3. Multiple conditions:
//     <foreach collection="ids" item="ID" separator="OR">
//     ID = #{ID}
//     </foreach>
//
// Example results:
//
//	Input collection: [1, 2, 3]
//	xmlConfiguration: open="(", separator=",", close=")"
//	Output: "(1,2,3)"
type ForeachNode struct {
	Collection string
	Nodes      []Node
	Item       string
	Index      string
	Open       string
	Close      string
	Separator  string
	BindNodes  BindNodeGroup
}

// Accept accepts parameters and returns query and arguments.
func (f ForeachNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	p = f.BindNodes.ConvertParameter(p)

	// if item already exists
	if _, exists := p.Get(f.Item); exists {
		return "", nil, fmt.Errorf("item %s already exists", f.Item)
	}

	// one collection from parameter
	value, exists := p.Get(f.Collection)
	if !exists {
		return "", nil, fmt.Errorf("collection %s not found", f.Collection)
	}

	// if valueItem can not be iterated
	if !value.CanInterface() {
		return "", nil, fmt.Errorf("collection %s can not be iterated", f.Collection)
	}

	// if valueItem is not a slice
	for value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Array, reflect.Slice:
		return f.acceptSlice(value, translator, p)
	case reflect.Map:
		return f.acceptMap(value, translator, p)
	default:
		return "", nil, fmt.Errorf("collection %s is not a slice or map", f.Collection)
	}
}

func (f ForeachNode) acceptSlice(value reflect.Value, translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	sliceLength := value.Len()

	if sliceLength == 0 {
		return "", nil, nil
	}

	// Pre-allocate args slice capacity to avoid multiple growths
	// Estimate: number of slice elements * number of Nodes
	estimatedArgsLen := sliceLength * len(f.Nodes)

	args = make([]any, 0, estimatedArgsLen)

	// Pre-allocate string builder capacity to minimize buffer reallocations
	// Capacity = open + items + separators + close
	estimatedBuilderCap := len(f.Open) + (2 * sliceLength) + (len(f.Separator) * (sliceLength - 1)) + len(f.Close)

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	builder.Grow(estimatedBuilderCap)

	builder.WriteString(f.Open)

	end := sliceLength - 1

	h := make(eval.H, 2)

	// Create and reuse GenericParameter outside the loop to avoid allocations per iteration
	genericParameter := &eval.GenericParameter{Value: reflect.ValueOf(h)}

	group := eval.ParamGroup{genericParameter, p}

	for i := 0; i < sliceLength; i++ {

		item := value.Index(i).Interface()

		h[f.Item] = item
		h[f.Index] = i

		for _, node := range f.Nodes {
			q, a, err := node.Accept(translator, group)
			if err != nil {
				return "", nil, err
			}
			if len(q) > 0 {
				builder.WriteString(q)
			}
			if len(a) > 0 {
				args = append(args, a...)
			}
		}

		if i < end {
			builder.WriteString(f.Separator)
		}
		genericParameter.Clear()
	}

	// if sliceLength is not zero, add close
	builder.WriteString(f.Close)

	return builder.String(), args, nil
}

func (f ForeachNode) acceptMap(value reflect.Value, translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	keys := value.MapKeys()

	if len(keys) == 0 {
		return "", nil, nil
	}

	// Pre-allocate args slice capacity to avoid multiple growths
	// Estimate: number of slice elements * number of Nodes
	estimatedArgsLen := len(keys) * len(f.Nodes)

	args = make([]any, 0, estimatedArgsLen)

	// Pre-allocate string builder capacity to minimize buffer reallocations
	// Capacity = open + items + separators + close
	estimatedBuilderCap := len(f.Open) + (2 * len(keys)) + (len(f.Separator) * (len(keys) - 1)) + len(f.Close)

	var builder = getStringBuilder()
	defer putStringBuilder(builder)

	builder.Grow(estimatedBuilderCap)

	builder.WriteString(f.Open)

	end := len(keys) - 1

	var index int

	h := make(eval.H, 2)

	// Create and reuse GenericParameter outside the loop to avoid allocations per iteration
	genericParameter := &eval.GenericParameter{Value: reflect.ValueOf(h)}

	group := eval.ParamGroup{genericParameter, p}

	for _, key := range keys {

		item := value.MapIndex(key).Interface()

		h[f.Item] = item
		h[f.Index] = key.Interface()

		for _, node := range f.Nodes {
			q, a, err := node.Accept(translator, group)
			if err != nil {
				return "", nil, err
			}
			if len(q) > 0 {
				builder.WriteString(q)
			}
			if len(a) > 0 {
				args = append(args, a...)
			}
		}

		if index < end {
			builder.WriteString(f.Separator)
		}

		genericParameter.Clear()

		index++
	}

	builder.WriteString(f.Close)

	return builder.String(), args, nil
}

var _ Node = (*ForeachNode)(nil)
