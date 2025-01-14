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

package eval

import (
	"context"
	"github.com/go-juicedev/juice/internal/reflectlite"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// Param is an alias of any type.
// It is used to represent the parameter of the xmlSQLStatement and without type limitation.
type Param = any

type paramCtxKey struct{}

// CtxWithParam returns a new context with the parameter.
func CtxWithParam(ctx context.Context, param Param) context.Context {
	return context.WithValue(ctx, paramCtxKey{}, param)
}

// ParamFromContext returns the parameter from the context.
func ParamFromContext(ctx context.Context) Param {
	param := ctx.Value(paramCtxKey{})
	return param
}

// defaultParamKey is the default key of the parameter.
var defaultParamKey = func() string {
	// try to get the key from environment variable
	key := os.Getenv("JUICE_PARAM_KEY")
	// if not found, use the default key
	if len(key) == 0 {
		key = "param"
	}
	return key
}()

// DefaultParamKey returns the default key of the parameter.
func DefaultParamKey() string {
	return defaultParamKey
}

// Parameter is the interface that wraps the Get method.
// Get returns the value of the named parameter.
type Parameter interface {
	// Get returns the value of the named parameter with the type of reflect.Value.
	Get(name string) (reflect.Value, bool)
}

// NoOPParameter is a no-op parameter.
// Its does nothing when calling the Get method.
type NoOPParameter struct{}

// Get implements Parameter.
// always return false.
func (NoOPParameter) Get(_ string) (reflect.Value, bool) {
	return reflect.Value{}, false
}

var noOPParameter Parameter = NoOPParameter{}

// make sure that ParamGroup implements Parameter.
var _ Parameter = (ParamGroup)(nil)

// ParamGroup is a group of parameters which implements the Parameter interface.
type ParamGroup []Parameter

// Get implements Parameter.
func (g ParamGroup) Get(name string) (reflect.Value, bool) {
	for _, p := range g {
		if p == nil {
			continue
		}
		if value, ok := p.Get(name); ok {
			return value, ok
		}
	}
	return reflect.Value{}, false
}

// make sure that structParameter implements Parameter.
var _ Parameter = (*structParameter)(nil)

// structParameter is a parameter that wraps a struct.
type structParameter struct {
	reflect.Value
	fieldIndexes map[string][]int
}

// Get implements Parameter.
func (p *structParameter) Get(name string) (reflect.Value, bool) {
	if len(name) == 0 {
		return reflect.Value{}, false
	}
	// Check type cache first
	if indexes, ok := p.fieldIndexes[name]; ok {
		return p.FieldByIndex(indexes), true
	}

	// if isPublic it means that the name is exported
	isPublic := unicode.IsUpper(rune(name[0]))
	var indexes []int
	if !isPublic {
		var ok bool
		// try to find the field by tag
		indexes, ok = reflectlite.TypeFrom(p.Value.Type()).GetFieldIndexesFromTag(defaultParamKey, name)
		if !ok {
			return reflect.Value{}, false
		}
	} else {
		// Find field index by name
		field, ok := p.Type().FieldByName(name)
		if !ok {
			return reflect.Value{}, false
		}
		indexes = field.Index
	}

	// Cache the field index for future use
	p.fieldIndexes[name] = indexes

	value := p.FieldByIndex(indexes)
	return value, value.IsValid()
}

// make sure that mapParameter implements Parameter.
var _ Parameter = (*mapParameter)(nil)

// mapParameter is a parameter that wraps a map.
type mapParameter struct {
	reflect.Value
}

// Get implements Parameter.
func (p mapParameter) Get(name string) (reflect.Value, bool) {
	value := p.MapIndex(reflect.ValueOf(name))
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	return value, true
}

// make sure that sliceParameter implements Parameter.
var _ Parameter = (*sliceParameter)(nil)

// sliceParameter is a parameter that wraps a slice.
type sliceParameter struct {
	reflect.Value
}

// Get implements Parameter.
func (p sliceParameter) Get(name string) (reflect.Value, bool) {
	index, err := strconv.Atoi(name)
	if err != nil {
		return reflect.Value{}, false
	}
	value := p.Index(index)
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	return value, true
}

// GenericParameter is a parameter that wraps a generic value.
type GenericParameter struct {
	// Value is the wrapped value
	Value reflect.Value

	// cache is used to cache the final value of the parameter path.
	// For example, if the path is "user.address.street",
	// it will cache the final street value to avoid parsing the path again.
	cache map[string]reflect.Value

	// structFieldIndex caches the field indexes for struct types at each path level.
	// The first key is the position in the path (e.g., for "user.address.street": 0 for user, 1 for address).
	// The second key is the concrete type of the struct, which ensures correct field lookup for different struct types.
	// The third key is the field name, and the value is the field index slice.
	// This three-level cache design ensures that field indexes are not mixed between different struct types,
	// which is particularly important when dealing with slices of different struct types.
	structFieldIndex map[int]map[reflect.Type]map[string][]int
}

func (g *GenericParameter) get(name string) (value reflect.Value, exists bool) {
	value = g.Value
	items := strings.Split(name, ".")
	var param Parameter
	for i, item := range items {

		// only unwrap when the value need to call Get method
		value = reflectlite.Unwrap(value)

		// match the value type
		// only map, struct, slice and array can be wrapped as parameter
		switch value.Kind() {
		case reflect.Map:
			// if the map key is not a string type, then return false
			if value.Type().Key().Kind() != reflect.String {
				return reflect.Value{}, false
			}
			param = mapParameter{Value: value}
		case reflect.Struct:
			// Initialize the three-level cache if not exists:
			// Level 1: path position -> to handle different levels in the path (e.g., user.address.street)
			// Level 2: concrete type -> to handle different struct types at the same position
			// Level 3: field name -> to cache the actual field indexes
			if g.structFieldIndex == nil {
				g.structFieldIndex = make(map[int]map[reflect.Type]map[string][]int)
			}
			
			// Cache the type to avoid multiple calls to Type()
			valueType := value.Type()
			
			// Get or create the type-level cache for current path position
			structFieldIndex, in := g.structFieldIndex[i]
			if !in {
				// Initialize with the current type to avoid another map lookup
				structFieldIndex = map[reflect.Type]map[string][]int{
					valueType: {},
				}
				g.structFieldIndex[i] = structFieldIndex
			}
			
			// Create a new structParameter with its field cache pointing to 
			// the cached indexes for its specific type, ensuring different 
			// struct types don't share the same field index cache
			param = &structParameter{Value: value, fieldIndexes: structFieldIndex[valueType]}
		case reflect.Slice, reflect.Array:
			param = sliceParameter{Value: value}
		default:
			// otherwise, return false
			return reflect.Value{}, false
		}
		value, exists = param.Get(item)
		if !exists {
			return reflect.Value{}, false
		}
	}
	return value, true
}

// Get implements Parameter.
// It will scopeCache the value of the parameter for better performance.
func (g *GenericParameter) Get(name string) (value reflect.Value, exists bool) {
	// try to get the value from scopeCache first
	value, exists = g.cache[name]
	if exists {
		return value, exists
	}
	// if not found, then get the value from the generic parameter
	value, exists = g.get(name)
	if exists {
		if g.cache == nil {
			g.cache = make(map[string]reflect.Value)
		}
		// scopeCache the value
		g.cache[name] = value
	}
	return value, exists
}

// Clear clears the cache of the parameter.
func (g *GenericParameter) Clear() {
	clear(g.cache)
}

// NewGenericParam creates a generic parameter.
// if the value is not a map, struct, slice or array, then wrap it as a map.
func NewGenericParam(v any, wrapKey string) Parameter {
	if v == nil {
		return noOPParameter
	}
	value := reflect.ValueOf(v)

	tp := reflectlite.IndirectType(value.Type())

	switch tp.Kind() {
	case reflect.Map, reflect.Struct:
		// do nothing
	default:
		// if the value is not a map, struct, slice or array, then wrap it as a map
		if wrapKey == "" {
			wrapKey = defaultParamKey
		}
		value = reflect.ValueOf(H{wrapKey: v})
	}
	return &GenericParameter{Value: value}
}

// NewParameter creates a new parameter with the given value.
func NewParameter(v Param) Parameter {
	return NewGenericParam(v, "")
}

// H is a shortcut for map[string]any
type H map[string]any

// AsParam converts the H to a Parameter.
func (h H) AsParam() Parameter {
	return NewParameter(h)
}
