/*
Copyright 2026 eatmoreapple

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
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
)

// ExprCompiler is an evaluator of the expression.
type ExprCompiler interface {
	// Compile compiles the expression and returns the expression.
	Compile(expr string) (Expression, error)
}

// staticExprOptimizer is used to optimize static expressions at compile time
type staticExprOptimizer struct{}

// isStaticExpr checks if an expression is static (does not depend on runtime values)
func (s *staticExprOptimizer) isStaticExpr(exp ast.Expr) bool {
	switch exp := exp.(type) {
	case *ast.BasicLit:
		return true
	case *ast.BinaryExpr:
		return s.isStaticExpr(exp.X) && s.isStaticExpr(exp.Y)
	case *ast.ParenExpr:
		return s.isStaticExpr(exp.X)
	case *ast.UnaryExpr:
		return s.isStaticExpr(exp.X)
	default:
		return false
	}
}

// Optimize optimizes static expressions by evaluating them at compile time
func (s *staticExprOptimizer) Optimize(exp ast.Expr, params Parameter) (ast.Expr, error) {
	if !s.isStaticExpr(exp) {
		return exp, nil
	}

	// Evaluate the static expression
	value, err := eval(exp, params)
	if err != nil {
		return exp, err
	}

	// Convert the evaluation result to the corresponding literal expression
	switch value.Kind() {
	case reflect.Bool:
		return &ast.Ident{Name: strconv.FormatBool(value.Bool())}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &ast.BasicLit{
			Kind:  token.INT,
			Value: strconv.FormatInt(value.Int(), 10),
		}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &ast.BasicLit{
			Kind:  token.INT,
			Value: strconv.FormatUint(value.Uint(), 10),
		}, nil
	case reflect.Float32, reflect.Float64:
		return &ast.BasicLit{
			Kind:  token.FLOAT,
			Value: strconv.FormatFloat(value.Float(), 'f', -1, 64),
		}, nil
	case reflect.String:
		return &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(value.String()),
		}, nil
	default:
		return exp, nil
	}
}

// goExprCompiler compiles expressions using Go's AST parser.
type goExprCompiler struct{}

// Compile parses and optimizes an expression.
func (e *goExprCompiler) Compile(expr string) (Expression, error) {
	// Convert logical aliases (and, or, not) to Go operators (&&, ||, !).
	lexer := NewLexer(expr)
	// Tokenize the expression while preserving non-operator tokens.
	expr = lexer.Tokenize()

	// Parse the processed expression into an AST that can be evaluated.
	exp, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, &SyntaxError{err}
	}

	// Optimize static expressions at compile time.
	// This optimization process:
	// 1. Evaluates expressions that don't depend on runtime values (e.g., "1 + 2", "true && false")
	// 2. Replaces the expression with its computed result as a literal
	// 3. Reduces runtime overhead by pre-computing constant expressions
	optimizer := &staticExprOptimizer{}
	optimizedExp, err := optimizer.Optimize(exp, nil)
	if err != nil {
		return nil, err
	}

	return &goExpression{Expr: optimizedExp}, nil
}

// goExpression evaluates a parsed Go AST expression.
type goExpression struct {
	ast.Expr
}

// Execute evaluates the expression and returns the value.
func (e *goExpression) Execute(params Parameter) (Value, error) {
	return eval(e.Expr, params)
}

// defaultCompiler is the default expression compiler used by the package.
var defaultCompiler ExprCompiler = &goExprCompiler{}

// WithCompiler sets the default expression compiler.
// nil is not allowed.
//
// It is not safe for concurrent use with Compile; it should be called
// during initialization, before any expression is compiled.
func WithCompiler(exprCompiler ExprCompiler) {
	if exprCompiler == nil {
		panic("exprCompiler cannot be nil")
	}
	defaultCompiler = exprCompiler
}

// Compile compiles the expression and returns the expression.
func Compile(expr string) (Expression, error) {
	return defaultCompiler.Compile(expr)
}

var _ ExprCompiler = (*goExprCompiler)(nil)
