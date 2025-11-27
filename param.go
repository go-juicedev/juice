package juice

import (
	"cmp"
	"context"
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
	// configuration may be used in the future for more complex parameter building.
	parameterKey := eval.DefaultParamKey()
	return eval.ParamGroup{
		// paramName attribute will be deprecated in the future versions.
		newGenericParam(param, cmp.Or(statement.Attribute("paramName"), parameterKey)),
		eval.H{
			"_databaseId": driverName,
			parameterKey:  param,
		},
	}
}
