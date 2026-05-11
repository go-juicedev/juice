package juice

import (
	"github.com/go-juicedev/juice/eval"
)

// H is an alias of eval.H.
type H = eval.H

// buildStatementParameters builds the statement parameters.
func buildStatementParameters(param any, statement Statement, driverName string, _ Configuration) eval.Parameter {
	// Configuration is reserved for future parameter-building options.
	parameter := eval.ParamGroup{
		eval.NewGenericParam(param, statement.Attribute("paramName")),

		// Internal parameters for transporting extra statement metadata.
		// User-defined parameters may override them.
		eval.H{
			"_databaseId": driverName,
		},
		// Compatibility alias for the original parameter.
		// map[string]User{"foo": {Name: "bar"}} => _parameter.foo.name
		// User{Name: "bar"} => _parameter.name
		eval.PrefixPatternParameter("_parameter", param),
	}

	return parameter
}
