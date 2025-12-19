/*
Copyright 2025 eatmoreapple

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
	"errors"
	"log"
	"reflect"

	"github.com/go-juicedev/juice/eval"
)

// BindNode represents a named bind variable backed by a compiled expression.
type BindNode struct {
	Name string
	expr eval.Expression
}

// Parse compiles the given expression string and stores the result.
func (b *BindNode) Parse(expression string) (err error) {
	b.expr, err = eval.Compile(expression)
	return err
}

// Execute evaluates the compiled expression against the provided Parameter
// and returns the resulting reflect.Value.
func (b *BindNode) Execute(p eval.Parameter) (reflect.Value, error) {
	value, err := b.expr.Execute(p)
	if err != nil {
		return reflect.Value{}, err
	}
	return value, nil
}

type BindNodeGroup []*BindNode

func (b BindNodeGroup) ConvertParameter(parameter eval.Parameter) eval.Parameter {
	if len(b) == 0 {
		return parameter
	}
	// decorate the parameter with boundParameterDecorator
	// to provide binding scope for bind variables
	boundParam := &boundParameterDecorator{
		scope: &bindScope{
			nodes:     b,
			parameter: parameter,
		},
	}

	parameter = eval.ParamGroup{
		boundParam,
		parameter,
	}
	// another approach is to use ParamGroup to combine boundParam and parameter
	// but the order matters here.
	// if we put boundParam after parameter, the boundParam will have lower priority
	// than the original parameter, which is not what we want.
	// so we put boundParam before parameter.
	return parameter
}

// ErrBindVariableNotFound is returned when a bind variable lookup fails.
var ErrBindVariableNotFound = errors.New("juice: bind variable not found")

type boundParameterDecorator struct {
	scope *bindScope
}

func (e boundParameterDecorator) Get(name string) (reflect.Value, bool) {
	value, err := e.scope.Get(name)
	if err != nil {
		// it means the bind variable is not found in the bind scope
		// should we handle this error differently?
		// or just ignore it and let the underlying parameter handle it?
		if !errors.Is(err, ErrBindVariableNotFound) {
			// just log it for debugging purpose
			log.Printf("[WARN] BindVariableNotFound when getting parameter %s: %v", name, err)
		}
		return reflect.Value{}, false
	}
	return value, true
}
