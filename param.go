package juice

import (
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
func buildStatementParameters(param any, statement Statement, driverName string, configuration Configuration) eval.Parameter {
	return eval.ParamGroup{
		newGenericParam(param, statement.Attribute("paramName")),

		// internal parameters for transporting extra information
		// those parameters may be overwritten by user-defined parameters
		eval.H{
			"_databaseId": driverName,
			"_parameter":  param,
		},
	}
}
