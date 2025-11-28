package juice

import (
	"context"
	"errors"
	"reflect"

	"github.com/go-juicedev/juice/eval"
)

// Param is an alias of eval.Param.
type Param = eval.Param

// Parameter is an alias of eval.Parameter.
type Parameter = eval.Parameter

// H is an alias of eval.H.
type H = eval.H

// ParamFromContext returns the parameter from the context.
func ParamFromContext(ctx context.Context) Param {
	return eval.ParamFromContext(ctx)
}

// CtxWithParam returns a new context with the parameter.
func CtxWithParam(ctx context.Context, param Param) context.Context {
	return eval.CtxWithParam(ctx, param)
}

// newGenericParam returns a new generic parameter.
func newGenericParam(v any, wrapKey string) Parameter {
	return eval.NewGenericParam(v, wrapKey)
}

// buildStatementParameters builds the statement parameters.
func buildStatementParameters(param any, statement Statement, driverName string, _ Configuration) eval.Parameter {
	// Configuration is not used currently, but kept for future extension for more complex parameter building logic
	parameter := eval.ParamGroup{
		newGenericParam(param, statement.Attribute("paramName")),

		// internal parameters for transporting extra information
		// those parameters may be overwritten by user-defined parameters
		eval.H{
			"_databaseId": driverName,
		},
		// this may cause something unexpected,
		// but I can not figure out.

		// map[string]User{"foo": {Name: "bar"}} => _parameter.foo.name // bar
		// User{Name: "bar"} => _parameter.name // bar
		eval.PrefixPatternParameter("_parameter", param),
	}

	if bindNodes := statement.BindNodes(); len(bindNodes) > 0 {
		// decorate the parameter with boundParameterDecorator
		// to provide binding scope for bind variables
		boundParam := &boundParameterDecorator{
			scope: &bindScope{
				nodes:     bindNodes,
				parameter: parameter,
			},
		}

		boundParameter := make(eval.ParamGroup, 0, len(parameter)+1)
		boundParameter = append(boundParameter, boundParam)
		parameter = append(boundParameter, parameter...)

		// another approach is to use ParamGroup to combine boundParam and parameter
		// but the order matters here.
		// if we put boundParam after parameter, the boundParam will have lower priority
		// than the original parameter, which is not what we want.
		// so we put boundParam before parameter.
	}

	return parameter
}

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
			logger.Printf("[WARN] BindVariableNotFound when getting parameter %s: %v", name, err)
		}
		return reflect.Value{}, false
	}
	return value, true
}
