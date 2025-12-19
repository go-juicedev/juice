package juice

import (
	"github.com/go-juicedev/juice/eval"
)

// H is an alias of eval.H.
type H = eval.H

// buildStatementParameters builds the statement parameters.
func buildStatementParameters(param any, statement Statement, driverName string, _ Configuration) eval.Parameter {
	// Configuration is not used currently, but kept for future extension for more complex parameter building logic
	parameter := eval.ParamGroup{
		eval.NewGenericParam(param, statement.Attribute("paramName")),

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

	return parameter
}
