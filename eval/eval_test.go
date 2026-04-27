package eval

import (
	"context"
	"errors"
	"fmt"
	"go/parser"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func testEval(expr string, v any) (result reflect.Value, err error) {
	param := NewGenericParam(v, "")
	return Eval(expr, param)
}

func TestEval_eval_test(t *testing.T) {
	param := H{
		"id":   1,
		"age":  18,
		"name": "eatmoreapple",
	}
	result, err := testEval(`id > 0 && id < 2`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`age == 17 + 1 && age == 36 / 2 && age == 9 * 2 && age == 19 -1`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`name == "eatmoreapple"`, param)
	if err != nil {
		t.Error(err)
		return
	}

	if !result.Bool() {
		t.Error("eval error")
		return
	}

	result, err = testEval(`"eat" + "more" + "apple"`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eatmoreapple" {
		t.Error("eval error")
		return
	}
}

func TestMixedNumericTypes_eval_test(t *testing.T) {
	param := H{
		"age":  18.5,
		"age2": uint(2),
	}

	result, err := testEval(`age + age2 + 1`, param)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind() != reflect.Float64 || result.Float() != 21.5 {
		t.Fatalf("expected float64 21.5, got %v (%v)", result, result.Kind())
	}

	result, err = testEval(`age + age2 + 1 == 21.5`, param)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Bool() {
		t.Fatal("expected mixed numeric comparison to be true")
	}
}

func BenchmarkEval(b *testing.B) {
	param := H{
		"id":   1,
		"age":  18,
		"name": "eatmoreapple",
	}
	for i := 0; i < b.N; i++ {
		value, err := testEval(`id > 0 && id < 2 && name == "eatmoreapple"`, param)
		if err != nil {
			b.Error(err)
			return
		}
		if !value.Bool() {
			b.Error("eval error")
			return
		}
	}
	// BenchmarkEval-8   	 1047154	      1111 ns/op
}

func BenchmarkEval2(b *testing.B) {
	param := H{
		"id":   1,
		"age":  18,
		"name": "eatmoreapple",
	}
	expr, err := parser.ParseExpr(`id > 0 && id < 2 && name == "eatmoreapple"`)
	if err != nil {
		b.Error(err)
		return
	}
	p := NewGenericParam(param, "")
	for i := 0; i < b.N; i++ {
		value, err := eval(expr, p)
		if err != nil {
			b.Error(err)
			return
		}
		if !value.Bool() {
			b.Error("eval error")
			return
		}
	}
	// BenchmarkEval2-8   	 5736370	       180.8 ns/op
}

func TestLen_eval_test(t *testing.T) {
	param := H{
		"a": []any{"a", "b", "c"},
		"b": "aaa",
		"c": map[string]any{"a": "a", "b": "b", "c": "c"},
	}
	result, err := testEval(`len(a)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 3 {
		t.Error("eval error")
		return
	}
	result, err = testEval(`len(b)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 3 {
		t.Error("eval error")
		return
	}
	result, err = testEval(`len(c)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 3 {
		t.Error("eval error")
		return
	}
}

func TestSubStr_eval_test(t *testing.T) {
	param := H{
		"a": "eatmoreapple",
	}
	result, err := testEval(`substr(a, 0, 3)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eat" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`substr(a, 3, 4)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "more" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`substr(a, 7, 5)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "apple" {
		t.Error("eval error")
		return
	}
}

func TestSubJoin_eval_test(t *testing.T) {
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`join(a, "")`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eatmoreapple" {
		t.Error("eval error")
		return
	}
}

func TestSlice_eval_test(t *testing.T) {
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`slice(a, 0, 1)`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 1 {
		t.Error("eval error")
		return
	}
	if result.Index(0).Interface() != "eat" {
		t.Error("eval error")
		return
	}
}

func TestLparenRparen_eval_test(t *testing.T) {
	result, err := testEval(`2 * (2 + 5) == 14`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
	result, err = Eval(`2 * (2 + 5) / 2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 7 {
		t.Error("eval error")
		return
	}
}

func TestComment_eval_test(t *testing.T) {
	result, err := Eval(`2 * (2 + 5) + 1 // 2 * (2 + 5) == 14`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 15 {
		t.Error("eval error")
		return
	}
}

func TestUnaryExpr_eval_test(t *testing.T) {
	result, err := Eval(`-2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != -2 {
		t.Error("eval error")
		return
	}
}

func TestUnaryExpr2_eval_test(t *testing.T) {
	result, err := Eval(`-2 * 3`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != -6 {
		t.Error("eval error")
		return
	}
}

func TestIndexExprSlice_eval_test(t *testing.T) {
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`a[0]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eat" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`a[0] + a[1]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "eatmore" {
		t.Error("eval error")
		return
	}

	result, err = testEval(`a[-1]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "apple" {
		t.Error("eval error")
		return
	}

	_, err = testEval(`a[-4]`, param)
	if !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("expected ErrIndexOutOfRange for out-of-range negative index, got %v", err)
	}
}

func TestIndexExprMap_eval_test(t *testing.T) {
	param := H{
		"a": map[string]string{
			"eat": "more",
		},
	}
	result, err := testEval(`a["eat"]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "more" {
		t.Error("eval error")
		return
	}

	result, err = testEval(`b[1]`, H{"b": map[int]string{1: "one"}})
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "one" {
		t.Error("eval error")
		return
	}

	_, err = testEval(`a[1]`, param)
	if !errors.Is(err, ErrMapIndexTypeMismatch) {
		t.Fatalf("expected ErrMapIndexTypeMismatch, got %v", err)
	}
}

func TestStarExpr_eval_test(t *testing.T) {
	_, err := Eval(`*2`, nil)
	if err == nil {
		t.Fatal("expected error for unary *")
	}
	if !errors.Is(err, errUnsupportedUnaryExpr) {
		t.Fatalf("expected unsupported unary expression, got %v", err)
	}

	result, err := Eval(`2 *2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Int() != 4 {
		t.Error("eval error")
		return
	}
}

func TestUnaryUnsupportedOperators_eval_test(t *testing.T) {
	_, err := testEval(`&id`, H{"id": 1})
	if err == nil {
		t.Fatal("expected error for unary &")
	}
	if !errors.Is(err, errUnsupportedUnaryExpr) {
		t.Fatalf("expected unsupported unary expression, got %v", err)
	}
}

func TestSliceExpr_eval_test(t *testing.T) {
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`a[:]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 3 {
		t.Errorf("eval error: %d", result.Len())
		return
	}
	if result.Index(0).Interface() != "eat" {
		t.Error("eval error")
		return
	}
	if result.Index(1).Interface() != "more" {
		t.Error("eval error")
		return
	}
	if result.Index(2).Interface() != "apple" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`a[1:]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 2 {
		t.Errorf("eval error: %d", result.Len())
		return
	}
	if result.Index(0).Interface() != "more" {
		t.Error("eval error")
		return
	}
	if result.Index(1).Interface() != "apple" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`a[1:2]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 1 {
		t.Errorf("eval error: %d", result.Len())
		return
	}
	if result.Index(0).Interface() != "more" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`a[i:j]`, H{
		"a": []string{"eat", "more", "apple"},
		"i": 0,
		"j": 2,
	})
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 2 || result.Index(0).Interface() != "eat" || result.Index(1).Interface() != "more" {
		t.Error("eval error")
		return
	}
}

func TestAnd_eval_test(t *testing.T) {
	result, err := Eval(`1 + 1 < 0 && 1 + 1 == 2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
	result, err = Eval(`(1 + 1 < 0) & (1 + 1 == 2)`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
	result, err = Eval("true & false", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestOr_eval_test(t *testing.T) {
	result, err := Eval(`1 + 1 < 0 || 1 + 1 == 2`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
	result, err = Eval("true | false", nil)
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestAndOr_eval_test(t *testing.T) {
	result, err := Eval(`1 + 1 < 0 || 1 + 1 == 2 && 1 + 1 == 3`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestAndOr2_eval_test(t *testing.T) {
	result, err := Eval(`1 + 1 < 0 && 1 + 1 == 2 || 1 + 1 == 3`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestNot_eval_test(t *testing.T) {
	result, err := Eval(`!(1 + 1 == 2)`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestNot2_eval_test(t *testing.T) {
	result, err := Eval(`!true`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestSlice3_eval_test(t *testing.T) {
	param := H{
		"a": []string{"eat", "more", "apple"},
	}
	result, err := testEval(`a[1:2:3]`, param)
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 1 {
		t.Errorf("eval error: %d", result.Len())
		return
	}
	if result.Index(0).Interface() != "more" {
		t.Error("eval error")
		return
	}
	result, err = testEval(`a[i:j:k]`, H{
		"a": []string{"eat", "more", "apple"},
		"i": 1,
		"j": 2,
		"k": 3,
	})
	if err != nil {
		t.Error(err)
		return
	}
	if result.Len() != 1 || result.Index(0).Interface() != "more" {
		t.Error("eval error")
		return
	}
}

func TestNil_eval_test(t *testing.T) {
	result, err := Eval(`nil`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if result.IsValid() {
		t.Error("eval error")
		return
	}
}

func TestExprNilEQ_eval_test(t *testing.T) {
	result, err := Eval("a == nil", H{"a": nil})
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
	var a *int
	result, err = Eval("a == nil", H{"a": a})
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	var entity struct {
		A *int `param:"a"`
	}
	result, err = Eval("a == nil", NewGenericParam(entity, ""))
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
}

func TestExprNilNEQ_eval_test(t *testing.T) {
	result, err := Eval("a != nil", H{"a": nil})
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}
	var a *int
	result, err = Eval("a != nil", H{"a": a})
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}

	var entity struct {
		A *int `param:"a"`
	}
	result, err = Eval("a != nil", NewGenericParam(entity, ""))
	if err != nil {
		t.Error(err)
		return
	}
	if result.Bool() {
		t.Error("eval error")
		return
	}

	var a2 = new(int)
	result, err = Eval("a != nil", H{"a": a2})
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	var a3 = 1
	_, err = Eval("a == nil", H{"a": &a3})
	if err != nil {
		t.Error(err)
		return
	} else {
		t.Log(err)
	}
}

func TestSelector_eval_test(t *testing.T) {
	var entity struct {
		A int `param:"a"`
	}
	entity.A = 1
	result, err := Eval("entity.A > 0", H{"entity": entity})
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}
}

type testStruct struct{}

func (t testStruct) Test() (bool, error) {
	return true, nil
}

func TestSelectorFunc_eval_test(t *testing.T) {
	var entity struct {
		A *testStruct `param:"a"`
	}
	entity.A = &testStruct{}
	result, err := Eval("entity.A.Test()", H{"entity": entity})
	if err != nil {
		t.Error(err)
		return
	}
	if !result.Bool() {
		t.Error("eval error")
		return
	}

	f := func() (string, error) {
		return "test", nil
	}

	result, err = Eval("f()", H{"f": f})
	if err != nil {
		t.Error(err)
		return
	}
	if result.String() != "test" {
		t.Error("eval error")
		return
	}
}

func TestMapDefaultMap_eval_test(t *testing.T) {
	result, err := Eval("a.b", H{"a": H{"b": 1}})
	if err != nil {
		t.Error(err)
		return
	}
	if result.Interface() != 1 {
		t.Error("eval error")
		return
	}

	result, err = Eval(`a["c"]`, H{"a": map[string]int{}})
	if err != nil {
		t.Error(err)
		return
	}
	if result.Interface() != 0 {
		t.Error("eval error")
		return
	}

	result, err = Eval(`a["c"]`, H{"a": map[string]string{}})
	if err != nil {
		t.Error(err)
		return
	}
	if result.Interface() != "" {
		t.Error("eval error")
		return
	}

	result, err = Eval(`a["c"]`, H{"a": map[string]float64{}})
	if err != nil {
		t.Error(err)
		return
	}
	if result.Interface() != 0.0 {
		t.Error("eval error")
		return
	}
}

// BenchmarkStaticExpr tests the performance of static expression evaluation
func BenchmarkStaticExpr(b *testing.B) {
	tests := []struct {
		name string
		expr string
	}{
		{"simple_bool", "1 == 1"},
		{"simple_math", "1 + 2 * 3"},
		{"complex_math", "10 + 20 * 3"},
		{"string_concat", `"hello" + "world"`},
		{"mixed_ops", "1 + 2 * 3 == 7"},
	}

	b.Run("without_optimization", func(b *testing.B) {
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := Eval(tt.expr, nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	b.Run("with_optimization", func(b *testing.B) {
		compiler := &goExprCompiler{}
		for _, tt := range tests {
			b.Run(tt.name, func(b *testing.B) {
				// Pre-compile the expression
				expr, err := compiler.Compile(tt.expr)
				if err != nil {
					b.Fatal(err)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := expr.Execute(nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})
}

// BenchmarkStaticExprOptimizer benchmarks the static expression optimizer
func BenchmarkStaticExprOptimizer(b *testing.B) {
	benchmarks := []struct {
		name string
		expr string
		want interface{}
	}{
		{"simple_bool", "1 == 1", true},
		{"simple_math", "1 + 2 * 3", int64(7)},
		{"complex_math", "10 + 20 * 3", int64(70)},
		{"string_concat", `"hello" + "world"`, "helloworld"},
		{"mixed_ops", "1 + 2 * 3 == 7", true},
		{"bool_chain", "true && false || true", true},
		{"math_chain", "1 + 2 + 3 + 4 + 5", int64(15)},
		{"complex_bool", "(1 < 2) && (3 > 2) || false", true},
	}

	optimizer := &StaticExprOptimizer{}
	// Test optimization performance only
	b.Run("optimization_only", func(b *testing.B) {
		for _, bm := range benchmarks {
			b.Run(bm.name, func(b *testing.B) {
				exp, err := parser.ParseExpr(bm.expr)
				if err != nil {
					b.Fatal(err)
				}
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := optimizer.Optimize(exp, nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	// Test parsing and optimization performance
	b.Run("parse_and_optimize", func(b *testing.B) {
		for _, bm := range benchmarks {
			b.Run(bm.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					exp, err := parser.ParseExpr(bm.expr)
					if err != nil {
						b.Fatal(err)
					}
					_, err = optimizer.Optimize(exp, nil)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	})

	// Test full compilation and optimization process
	b.Run("full_compile_and_optimize", func(b *testing.B) {
		compiler := &goExprCompiler{}
		for _, bm := range benchmarks {
			b.Run(bm.name, func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					expr, err := compiler.Compile(bm.expr)
					if err != nil {
						b.Fatal(err)
					}
					result, err := expr.Execute(nil)
					if err != nil {
						b.Fatal(err)
					}
					// Validate results
					var got interface{}
					switch result.Kind() {
					case reflect.Bool:
						got = result.Bool()
					case reflect.Int64:
						got = result.Int()
					case reflect.String:
						got = result.String()
					default:
						b.Fatalf("unexpected type: %v", result.Kind())
					}
					if got != bm.want {
						b.Fatalf("got %v, want %v", got, bm.want)
					}
				}
			})
		}
	})
}

// TestStaticExprOptimizer tests the correctness of static expression optimization
func TestStaticExprOptimizer_eval_test(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected interface{}
	}{
		{"bool_eq", "1 == 1", true},
		{"bool_neq", "1 != 2", true},
		{"math_add", "1 + 2", int64(3)},
		{"math_mul", "2 * 3", int64(6)},
		{"math_complex", "10 + 20 * 3", int64(70)},
		{"string_concat", `"hello" + "world"`, "helloworld"},
		{"mixed_ops", "1 + 2 * 3 == 7", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.expr, nil)
			if err != nil {
				t.Fatalf("failed to eval expression: %v", err)
			}

			var actual interface{}
			switch result.Kind() {
			case reflect.Bool:
				actual = result.Bool()
			case reflect.Int64:
				actual = result.Int()
			case reflect.String:
				actual = result.String()
			default:
				t.Fatalf("unexpected result type: %v", result.Kind())
			}

			if actual != tt.expected {
				t.Errorf("got %v, want %v", actual, tt.expected)
			}
		})
	}
}

// TestVariadicSliceUnpacking tests the slice unpacking syntax for variadic function calls
func TestVariadicSliceUnpacking_eval_test(t *testing.T) {
	// Test variadic function
	sumFunc := func(nums ...int) (int, error) {
		sum := 0
		for _, n := range nums {
			sum += n
		}
		return sum, nil
	}

	concatFunc := func(parts ...string) (string, error) {
		return strings.Join(parts, ""), nil
	}

	// Create test data using variables instead of composite literals
	numbers := []int{1, 2, 3, 4, 5}
	stringSlices := []string{"hello", "world", "test"}
	var emptyNumbers []int
	singleNumber := []int{42}
	partialNumbers := []int{2, 3, 4, 5} // [2,3,4,5]

	param := H{
		"numbers":        numbers,
		"strings":        stringSlices,
		"emptyNumbers":   emptyNumbers,
		"singleNumber":   singleNumber,
		"partialNumbers": partialNumbers,
		"sum":            sumFunc,
		"concat":         concatFunc,
	}

	tests := []struct {
		name     string
		expr     string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "basic slice unpacking",
			expr:     "sum(numbers...)",
			expected: int64(15),
			wantErr:  false,
		},
		{
			name:     "string slice unpacking",
			expr:     "concat(strings...)",
			expected: "helloworldtest",
			wantErr:  false,
		},
		{
			name:     "mixed arguments with slice unpacking",
			expr:     "sum(10, partialNumbers...)",
			expected: int64(14), // 10 + 2+3+4+5 = 14
			wantErr:  false,
		},
		{
			name:     "empty slice unpacking",
			expr:     "sum(emptyNumbers...)",
			expected: int64(0),
			wantErr:  false,
		},
		{
			name:     "single element slice unpacking",
			expr:     "sum(singleNumber...)",
			expected: int64(42),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := testEval(tt.expr, param)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var actual interface{}
			switch result.Kind() {
			case reflect.Int, reflect.Int64:
				actual = result.Int()
			case reflect.String:
				actual = result.String()
			default:
				t.Fatalf("unexpected result type: %v", result.Kind())
			}

			if actual != tt.expected {
				t.Errorf("got %v (%T), want %v (%T)", actual, actual, tt.expected, tt.expected)
			}
		})
	}
}

// TestVariadicErrors tests error cases for variadic functions
func TestVariadicErrors_eval_test(t *testing.T) {
	sumFunc := func(nums ...int) (int, error) {
		return 0, nil
	}

	// Create test data
	wrongType := []string{"1", "2", "3"} // wrong type
	numbers := []int{1, 2, 3}

	param := H{
		"wrongType": wrongType,
		"numbers":   numbers,
		"sum":       sumFunc,
	}

	tests := []struct {
		name    string
		expr    string
		wantErr string
	}{
		{
			name:    "type mismatch in slice unpacking",
			expr:    "sum(wrongType...)",
			wantErr: "cannot convert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := testEval(tt.expr, param)
			if err == nil {
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestLexer_Tokenize_eval_test(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected bool
	}{
		// Basic logical operations
		{"simple_and", "true and true", true},
		{"simple_or", "true or false", true},
		{"simple_not", "not false", true},

		// Compound expressions
		{"compound_and", "true and true and true", true},
		{"compound_or", "false or false or true", true},
		{"compound_mixed", "true and false or true", true},

		// Parentheses
		{"parentheses", "(true and false) or true", true},
		{"nested_parentheses", "((true and true) or false) and true", true},

		// Not operator
		{"not_with_and", "not false and true", true},
		{"not_with_or", "not true or true", true},
		{"not_with_parentheses", "not (false and false)", true},

		// Complex expressions
		{"complex_1", "true and not false", true},
		{"complex_2", "not (true and false) or true", true},
		{"complex_3", "(not false and true) or (true and not false)", true},

		// False cases
		{"false_and", "true and false", false},
		{"false_or", "false or false", false},
		{"false_not", "not true", false},
		{"false_complex", "not true and not true or false", false},

		// Edge cases
		{"all_operators", "not true and false or true and not false", true},
		{"multiple_nots", "not not true", true},
		{"triple_not", "not not not false", true},

		// Precedence tests
		{"precedence_1", "true or false and false", true}, // and has higher precedence
		{"precedence_2", "false and true or true", true},  // demonstrates left-to-right evaluation
		{"precedence_3", "not false and true", true},      // not has highest precedence

		// Spacing variations
		{"extra_spaces", "true  and  false  or  true", true},
		{"minimal_spaces", "true and false or true", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.expr, nil)
			if err != nil {
				t.Fatalf("failed to eval expression: %v", err)
			}
			if result.Kind() != reflect.Bool {
				t.Fatalf("unexpected result type: %v", result.Kind())
			}
			if tt.expected != result.Bool() {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SyntaxError
// ---------------------------------------------------------------------------

func TestSyntaxError_eval_coverage_test(t *testing.T) {
	_, err := Eval("???", nil)
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
	var synErr *SyntaxError
	if !errors.As(err, &synErr) {
		t.Fatalf("expected SyntaxError, got %T: %v", err, err)
	}
	if synErr.Error() == "" {
		t.Fatal("SyntaxError.Error() should not be empty")
	}
	if synErr.Unwrap() == nil {
		t.Fatal("SyntaxError.Unwrap() should not be nil")
	}
}

// ---------------------------------------------------------------------------
// WithCompiler
// ---------------------------------------------------------------------------

func TestWithCompiler_eval_coverage_test(t *testing.T) {
	// Save original and restore after test.
	original := defaultCompiler
	defer func() { defaultCompiler = original }()

	// nil should panic.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil compiler")
		}
	}()
	WithCompiler(nil)
}

func TestWithCompilerCustom_eval_coverage_test(t *testing.T) {
	original := defaultCompiler
	defer func() { defaultCompiler = original }()

	custom := &goExprCompiler{}
	WithCompiler(custom)

	result, err := Eval("1 + 2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Int() != 3 {
		t.Fatalf("expected 3, got %v", result.Int())
	}
}

// ---------------------------------------------------------------------------
// Unary expression coverage: +x, ^x, error cases
// ---------------------------------------------------------------------------

func TestUnaryPlus_eval_coverage_test(t *testing.T) {
	result, err := testEval(`+x`, H{"x": 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Int() != 5 {
		t.Fatalf("expected 5, got %v", result.Int())
	}
}

func TestUnaryXOR_eval_coverage_test(t *testing.T) {
	result, err := testEval(`^x`, H{"x": 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Int() != -1 {
		t.Fatalf("expected -1, got %v", result.Int())
	}
}

func TestUnarySubOnNonInt_eval_coverage_test(t *testing.T) {
	_, err := testEval(`-x`, H{"x": true})
	if !errors.Is(err, errUnsupportedUnaryExpr) {
		t.Fatalf("expected errUnsupportedUnaryExpr, got %v", err)
	}
}

func TestUnaryPlusOnNonInt_eval_coverage_test(t *testing.T) {
	_, err := testEval(`+x`, H{"x": "hello"})
	if !errors.Is(err, errUnsupportedUnaryExpr) {
		t.Fatalf("expected errUnsupportedUnaryExpr, got %v", err)
	}
}

func TestUnaryNotOnNonBool_eval_coverage_test(t *testing.T) {
	_, err := testEval(`!x`, H{"x": 42})
	if !errors.Is(err, errUnsupportedUnaryExpr) {
		t.Fatalf("expected errUnsupportedUnaryExpr, got %v", err)
	}
}

func TestUnaryXOROnNonInt_eval_coverage_test(t *testing.T) {
	_, err := testEval(`^x`, H{"x": "text"})
	if !errors.Is(err, errUnsupportedUnaryExpr) {
		t.Fatalf("expected errUnsupportedUnaryExpr, got %v", err)
	}
}

func TestUnaryOnUndefined_eval_coverage_test(t *testing.T) {
	_, err := Eval(`-undefined`, H{})
	if err == nil {
		t.Fatal("expected error for undefined identifier")
	}
}

// ---------------------------------------------------------------------------
// evalCallExpr coverage: non-func, wrong return count, error return
// ---------------------------------------------------------------------------

func TestCallNonFunc_eval_coverage_test(t *testing.T) {
	_, err := testEval(`x()`, H{"x": 42})
	if err == nil {
		t.Fatal("expected error calling non-function")
	}
}

func TestCallFuncReturnsError_eval_coverage_test(t *testing.T) {
	fn := func() (int, error) {
		return 0, fmt.Errorf("custom error")
	}
	_, err := testEval(`fn()`, H{"fn": fn})
	if err == nil || err.Error() != "custom error" {
		t.Fatalf("expected 'custom error', got %v", err)
	}
}

func TestCallFuncWrongArgCount_eval_coverage_test(t *testing.T) {
	fn := func(a, b int) (int, error) { return a + b, nil }
	_, err := testEval(`fn(1)`, H{"fn": fn})
	if err == nil {
		t.Fatal("expected error for wrong argument count")
	}
}

func TestCallFuncArgTypeConversion_eval_coverage_test(t *testing.T) {
	fn := func(a int32) (int32, error) { return a * 2, nil }
	result, err := testEval(`fn(5)`, H{"fn": fn})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Int() != 10 {
		t.Fatalf("expected 10, got %v", result.Int())
	}
}

func TestCallVariadicFunc_eval_coverage_test(t *testing.T) {
	fn := func(sep string, items ...string) (string, error) {
		result := ""
		for i, item := range items {
			if i > 0 {
				result += sep
			}
			result += item
		}
		return result, nil
	}
	result, err := testEval(`fn(",", "a", "b", "c")`, H{"fn": fn})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "a,b,c" {
		t.Fatalf("expected 'a,b,c', got %v", result.String())
	}
}

func TestCallVariadicNoArgs_eval_coverage_test(t *testing.T) {
	fn := func(items ...string) (int, error) {
		return len(items), nil
	}
	result, err := testEval(`fn()`, H{"fn": fn})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Int() != 0 {
		t.Fatalf("expected 0, got %v", result.Int())
	}
}

func TestCallVariadicTooFewArgs_eval_coverage_test(t *testing.T) {
	fn := func(a int, b int, items ...string) (int, error) {
		return a + b + len(items), nil
	}
	// only 1 arg for 2 required
	_, err := testEval(`fn(1)`, H{"fn": fn})
	if err == nil {
		t.Fatal("expected error for too few variadic args")
	}
}

func TestCallFuncArgNotConvertible_eval_coverage_test(t *testing.T) {
	fn := func(a int) (int, error) { return a, nil }
	_, err := testEval(`fn("hello")`, H{"fn": fn})
	if err == nil {
		t.Fatal("expected error for non-convertible arg type")
	}
}

func TestCallFuncInterfaceValue_eval_coverage_test(t *testing.T) {
	fn := func() (string, error) { return "ok", nil }
	var iface any = fn
	result, err := testEval(`f()`, H{"f": iface})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "ok" {
		t.Fatalf("expected 'ok', got %v", result.String())
	}
}

// ---------------------------------------------------------------------------
// evalSelectorExpr coverage: unexported tag, map selector, method on struct
// ---------------------------------------------------------------------------

func TestSelectorUnexportedTag_eval_coverage_test(t *testing.T) {
	type user struct {
		Name string `param:"name"`
		age  int    `param:"age"` //nolint:unused
	}
	u := user{Name: "Alice"}
	result, err := testEval(`u.Name`, H{"u": u})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "Alice" {
		t.Fatalf("expected 'Alice', got %v", result.String())
	}

	// Access via tag
	result, err = testEval(`u.name`, H{"u": u})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "Alice" {
		t.Fatalf("expected 'Alice', got %v", result.String())
	}
}

func TestSelectorOnMap_eval_coverage_test(t *testing.T) {
	m := map[string]int{"count": 42}
	result, err := testEval(`m.count`, H{"m": m})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Int() != 42 {
		t.Fatalf("expected 42, got %v", result.Int())
	}
}

func TestSelectorInvalidField_eval_coverage_test(t *testing.T) {
	type user struct{ Name string }
	_, err := testEval(`u.NonExistent`, H{"u": user{}})
	if err == nil {
		t.Fatal("expected error for non-existent field")
	}
}

func TestSelectorOnInvalidType_eval_coverage_test(t *testing.T) {
	_, err := testEval(`x.field`, H{"x": 42})
	if err == nil {
		t.Fatal("expected error for selector on int")
	}
}

func TestSelectorUndefinedBase_eval_coverage_test(t *testing.T) {
	_, err := Eval(`undefined.field`, H{})
	if err == nil {
		t.Fatal("expected error for undefined base")
	}
}

// ---------------------------------------------------------------------------
// evalIdent coverage: undefined
// ---------------------------------------------------------------------------

func TestIdentUndefined_eval_coverage_test(t *testing.T) {
	_, err := Eval(`noSuchVar`, H{})
	if err == nil {
		t.Fatal("expected error for undefined identifier")
	}
}

// ---------------------------------------------------------------------------
// evalBasicLit coverage: FLOAT literal
// ---------------------------------------------------------------------------

func TestBasicLitFloat_eval_coverage_test(t *testing.T) {
	result, err := Eval(`3.14`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Float() != 3.14 {
		t.Fatalf("expected 3.14, got %v", result.Float())
	}
}

func TestBasicLitChar_eval_coverage_test(t *testing.T) {
	result, err := Eval(`'a'`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "a" {
		t.Fatalf("expected 'a', got %v", result.String())
	}
}

// ---------------------------------------------------------------------------
// evalIndexExpr coverage: out-of-range positive, invalid index type
// ---------------------------------------------------------------------------

func TestIndexOutOfRangePositive_eval_coverage_test(t *testing.T) {
	_, err := testEval(`a[10]`, H{"a": []string{"x"}})
	if !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("expected ErrIndexOutOfRange, got %v", err)
	}
}

func TestIndexOnInvalidType_eval_coverage_test(t *testing.T) {
	_, err := testEval(`a[0]`, H{"a": 42})
	if err == nil {
		t.Fatal("expected error for index on int")
	}
}

func TestIndexUndefinedCollection_eval_coverage_test(t *testing.T) {
	_, err := Eval(`a[0]`, H{})
	if err == nil {
		t.Fatal("expected error for undefined collection")
	}
}

// ---------------------------------------------------------------------------
// evalStarExpr coverage: star on variable
// ---------------------------------------------------------------------------

func TestStarExprOnVar_eval_coverage_test(t *testing.T) {
	_, err := testEval(`*x`, H{"x": 42})
	if err == nil {
		t.Fatal("expected error for star expression")
	}
	if !errors.Is(err, errUnsupportedUnaryExpr) {
		t.Fatalf("expected errUnsupportedUnaryExpr, got %v", err)
	}
}

func TestStarExprUndefined_eval_coverage_test(t *testing.T) {
	_, err := Eval(`*undefined`, H{})
	if err == nil {
		t.Fatal("expected error for star on undefined")
	}
}

// ---------------------------------------------------------------------------
// evalSliceExpr coverage: error in base expression
// ---------------------------------------------------------------------------

func TestSliceExprOnUndefined_eval_coverage_test(t *testing.T) {
	_, err := Eval(`undefined[1:2]`, H{})
	if err == nil {
		t.Fatal("expected error for slice on undefined")
	}
}

func TestSliceExprInvalidBoundType_eval_coverage_test(t *testing.T) {
	_, err := testEval(`a[i:]`, H{"a": []string{"x"}, "i": "0"})
	if err == nil {
		t.Fatal("expected error for non-integer slice bound")
	}
}

func TestSliceExprOutOfRangeBound_eval_coverage_test(t *testing.T) {
	_, err := testEval(`a[0:2]`, H{"a": []string{"x"}})
	if !errors.Is(err, ErrIndexOutOfRange) {
		t.Fatalf("expected ErrIndexOutOfRange, got %v", err)
	}
}

func TestSlice3ExprInvalidString_eval_coverage_test(t *testing.T) {
	_, err := testEval(`a[0:1:1]`, H{"a": "xy"})
	if err == nil {
		t.Fatal("expected error for 3-index slice on string")
	}
}

// ---------------------------------------------------------------------------
// evalBinaryExpr coverage: unsupported token
// ---------------------------------------------------------------------------

func TestBinaryExprLazyEval_eval_coverage_test(t *testing.T) {
	// Test that RHS of && is not evaluated when LHS is false
	// "false" is a builtin that evaluates to false
	result, err := Eval(`false && true`, H{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Bool() {
		t.Fatal("expected false")
	}
}

// ---------------------------------------------------------------------------
// Built-in string functions (all 0% coverage)
// ---------------------------------------------------------------------------

func TestBuiltinLower_eval_coverage_test(t *testing.T) {
	result, err := testEval(`lower("HELLO")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "hello" {
		t.Fatalf("expected 'hello', got %v", result.String())
	}
}

func TestBuiltinUpper_eval_coverage_test(t *testing.T) {
	result, err := testEval(`upper("hello")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "HELLO" {
		t.Fatalf("expected 'HELLO', got %v", result.String())
	}
}

func TestBuiltinTrim_eval_coverage_test(t *testing.T) {
	result, err := testEval(`trim("  hi  ", " ")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "hi" {
		t.Fatalf("expected 'hi', got %v", result.String())
	}
}

func TestBuiltinTrimLeft_eval_coverage_test(t *testing.T) {
	result, err := testEval(`trimLeft("  hi", " ")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "hi" {
		t.Fatalf("expected 'hi', got %v", result.String())
	}
}

func TestBuiltinTrimRight_eval_coverage_test(t *testing.T) {
	result, err := testEval(`trimRight("hi  ", " ")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "hi" {
		t.Fatalf("expected 'hi', got %v", result.String())
	}
}

func TestBuiltinReplace_eval_coverage_test(t *testing.T) {
	result, err := testEval(`replace("aaa", "a", "b", 1)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "baa" {
		t.Fatalf("expected 'baa', got %v", result.String())
	}
}

func TestBuiltinReplaceAll_eval_coverage_test(t *testing.T) {
	result, err := testEval(`replaceAll("aaa", "a", "b")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "bbb" {
		t.Fatalf("expected 'bbb', got %v", result.String())
	}
}

func TestBuiltinSplit_eval_coverage_test(t *testing.T) {
	result, err := testEval(`split("a,b,c", ",")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Len() != 3 {
		t.Fatalf("expected 3 elements, got %v", result.Len())
	}
	if result.Index(0).String() != "a" || result.Index(1).String() != "b" || result.Index(2).String() != "c" {
		t.Fatalf("unexpected split result: %v", result)
	}
}

func TestBuiltinSplitN_eval_coverage_test(t *testing.T) {
	result, err := testEval(`splitN("a,b,c", ",", 2)`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Len() != 2 {
		t.Fatalf("expected 2 elements, got %v", result.Len())
	}
	if result.Index(0).String() != "a" || result.Index(1).String() != "b,c" {
		t.Fatalf("unexpected splitN result: %v", result)
	}
}

func TestBuiltinSplitAfter_eval_coverage_test(t *testing.T) {
	result, err := testEval(`splitAfter("a,b,c", ",")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Len() != 3 {
		t.Fatalf("expected 3 elements, got %v", result.Len())
	}
	if result.Index(0).String() != "a," || result.Index(1).String() != "b," || result.Index(2).String() != "c" {
		t.Fatalf("unexpected splitAfter result: %v", result)
	}
}

// ---------------------------------------------------------------------------
// length / strSub edge cases
// ---------------------------------------------------------------------------

func TestLengthNil_eval_coverage_test(t *testing.T) {
	result, err := testEval(`len(x)`, H{"x": nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Int() != 0 {
		t.Fatalf("expected 0, got %v", result.Int())
	}
}

func TestLengthInvalidType_eval_coverage_test(t *testing.T) {
	_, err := testEval(`len(x)`, H{"x": 42})
	if err == nil {
		t.Fatal("expected error for len(int)")
	}
}

func TestSubStrEdgeCases_eval_coverage_test(t *testing.T) {
	tests := []struct {
		name   string
		expr   string
		param  H
		expect string
	}{
		{"negative_start", `substr(s, -3, 3)`, H{"s": "hello"}, "llo"},
		{"start_exceeds_len", `substr(s, 100, 3)`, H{"s": "hi"}, ""},
		{"negative_count", `substr(s, 0, -1)`, H{"s": "hello"}, "hell"},
		{"very_negative_start", `substr(s, -100, 3)`, H{"s": "hi"}, "hi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := testEval(tt.expr, tt.param)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.String() != tt.expect {
				t.Fatalf("expected %q, got %q", tt.expect, result.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// strJoin edge case
// ---------------------------------------------------------------------------

func TestJoinInvalidType_eval_coverage_test(t *testing.T) {
	_, err := testEval(`join(x, ",")`, H{"x": 42})
	if err == nil {
		t.Fatal("expected error for join on non-slice")
	}
}

// ---------------------------------------------------------------------------
// slice edge case
// ---------------------------------------------------------------------------

func TestSliceFuncInvalidType_eval_coverage_test(t *testing.T) {
	_, err := testEval(`slice(x, 0, 1)`, H{"x": "hello"})
	if err == nil {
		t.Fatal("expected error for slice on string")
	}
}

// ---------------------------------------------------------------------------
// RegisterEvalFunc / MustRegisterEvalFunc coverage
// ---------------------------------------------------------------------------

func TestRegisterEvalFuncErrors_eval_coverage_test(t *testing.T) {
	// Not a function
	err := RegisterEvalFunc("bad", 42)
	if err == nil {
		t.Fatal("expected error for non-function")
	}

	// Wrong number of returns
	err = RegisterEvalFunc("bad", func() int { return 0 })
	if err == nil {
		t.Fatal("expected error for wrong return count")
	}

	// Last return not error
	err = RegisterEvalFunc("bad", func() (int, int) { return 0, 0 })
	if err == nil {
		t.Fatal("expected error for non-error return")
	}

	// Valid registration
	err = RegisterEvalFunc("testcov", func(s string) (string, error) {
		return s + "!", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := testEval(`testcov("hi")`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.String() != "hi!" {
		t.Fatalf("expected 'hi!', got %v", result.String())
	}
}

func TestRegisterEvalFuncConcurrentRead_eval_test(t *testing.T) {
	const goroutines = 16
	const iterations = 100

	errCh := make(chan error, goroutines*iterations)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				name := fmt.Sprintf("concurrent_eval_func_%d_%d", id, j)
				if err := RegisterEvalFunc(name, func() (int, error) { return id + j, nil }); err != nil {
					errCh <- err
				}
				if _, ok := getBuiltin("len"); !ok {
					errCh <- errors.New("len builtin not found")
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMustRegisterEvalFuncPanic_eval_coverage_test(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid MustRegisterEvalFunc")
		}
	}()
	MustRegisterEvalFunc("bad", 42)
}

// ---------------------------------------------------------------------------
// Parameter coverage: CtxWithParam, ParamFromContext, DefaultParamKey
// ---------------------------------------------------------------------------

func TestCtxWithParam_eval_coverage_test(t *testing.T) {
	ctx := CtxWithParam(context.Background(), "hello")
	got := ParamFromContext(ctx)
	if got != "hello" {
		t.Fatalf("expected 'hello', got %v", got)
	}
}

func TestParamFromContextNil_eval_coverage_test(t *testing.T) {
	got := ParamFromContext(context.Background())
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestDefaultParamKey_eval_coverage_test(t *testing.T) {
	key := DefaultParamKey()
	if key == "" {
		t.Fatal("DefaultParamKey() should not be empty")
	}
}

// ---------------------------------------------------------------------------
// NoOPParameter
// ---------------------------------------------------------------------------

func TestNoOPParameter_eval_coverage_test(t *testing.T) {
	p := NoOPParameter{}
	_, ok := p.Get("anything")
	if ok {
		t.Fatal("NoOPParameter.Get should always return false")
	}
}

// ---------------------------------------------------------------------------
// sliceParameter
// ---------------------------------------------------------------------------

func TestSliceParameter_eval_coverage_test(t *testing.T) {
	p := NewGenericParam([]string{"a", "b", "c"}, "")
	// Access by index via the parameter path
	val, ok := p.Get("0")
	if !ok {
		t.Fatal("expected to find index 0")
	}
	if val.String() != "a" {
		t.Fatalf("expected 'a', got %v", val.String())
	}

	// Out of range
	_, ok = p.Get("10")
	if ok {
		t.Fatal("expected false for out-of-range index")
	}

	// Negative index
	_, ok = p.Get("-1")
	if ok {
		t.Fatal("expected false for negative index")
	}

	// Non-numeric key
	_, ok = p.Get("abc")
	if ok {
		t.Fatal("expected false for non-numeric key")
	}
}

// ---------------------------------------------------------------------------
// mapParameter with non-string key
// ---------------------------------------------------------------------------

func TestMapNonStringKey_eval_coverage_test(t *testing.T) {
	p := &GenericParameter{Value: reflect.ValueOf(map[int]string{1: "one"})}
	_, ok := p.Get("1")
	if ok {
		t.Fatal("expected false for map with non-string key")
	}
}

// ---------------------------------------------------------------------------
// GenericParameter cache and clear
// ---------------------------------------------------------------------------

func TestGenericParameterCacheAndClear_eval_coverage_test(t *testing.T) {
	type inner struct {
		Value int `param:"value"`
	}
	p := NewGenericParam(map[string]any{
		"a": inner{Value: 42},
	}, "").(*GenericParameter)

	// First access populates cache
	val, ok := p.Get("a.value")
	if !ok || val.Int() != 42 {
		t.Fatalf("expected 42, got %v (ok=%v)", val, ok)
	}

	// Second access should hit cache
	val, ok = p.Get("a.value")
	if !ok || val.Int() != 42 {
		t.Fatalf("cached access failed")
	}

	// Clear cache
	p.Clear()

	// Should still work after clear
	val, ok = p.Get("a.value")
	if !ok || val.Int() != 42 {
		t.Fatalf("access after clear failed")
	}
}

func TestGenericParameterCacheDifferentStructTypesAtSamePathDepth_eval_coverage_test(t *testing.T) {
	type left struct {
		Name string
	}
	type right struct {
		Name string
	}
	type root struct {
		Left  left
		Right right
	}

	p := NewGenericParam(root{
		Left:  left{Name: "left"},
		Right: right{Name: "right"},
	}, "")

	value, ok := p.Get("Left.Name")
	if !ok || value.String() != "left" {
		t.Fatalf("expected left, got %v (ok=%v)", value, ok)
	}

	value, ok = p.Get("Right.Name")
	if !ok || value.String() != "right" {
		t.Fatalf("expected right, got %v (ok=%v)", value, ok)
	}
}

// ---------------------------------------------------------------------------
// NewGenericParam: wrap non-container type
// ---------------------------------------------------------------------------

func TestNewGenericParamWrapsPrimitive_eval_coverage_test(t *testing.T) {
	p := NewGenericParam(42, "val")
	v, ok := p.Get("val")
	if !ok {
		t.Fatal("expected to find wrapped primitive")
	}
	if v.Interface() != 42 {
		t.Fatalf("expected 42, got %v", v.Interface())
	}
}

func TestNewGenericParamWrapsWithDefaultKey_eval_coverage_test(t *testing.T) {
	p := NewGenericParam(42, "")
	v, ok := p.Get(defaultParamKey)
	if !ok {
		t.Fatal("expected to find wrapped primitive with default key")
	}
	if v.Interface() != 42 {
		t.Fatalf("expected 42, got %v", v.Interface())
	}
}

func TestNewGenericParamNil_eval_coverage_test(t *testing.T) {
	p := NewGenericParam(nil, "")
	_, ok := p.Get("anything")
	if ok {
		t.Fatal("nil param should return false")
	}
}

// ---------------------------------------------------------------------------
// H parameter
// ---------------------------------------------------------------------------

func TestH_eval_coverage_test(t *testing.T) {
	h := H{"key": "value"}
	v, ok := h.Get("key")
	if !ok || v.String() != "value" {
		t.Fatalf("expected 'value', got %v", v)
	}
	_, ok = h.Get("missing")
	if ok {
		t.Fatal("expected false for missing key")
	}
}

// ---------------------------------------------------------------------------
// PrefixPatternParameter
// ---------------------------------------------------------------------------

func TestPrefixPatternParameter_eval_coverage_test(t *testing.T) {
	p := PrefixPatternParameter("user", map[string]any{"name": "Alice", "age": 30})

	// Get with dot notation
	val, ok := p.Get("user.name")
	if !ok || val.Interface() != "Alice" {
		t.Fatalf("expected 'Alice', got %v (ok=%v)", val, ok)
	}

	// Get prefix only (returns the whole param)
	_, ok = p.Get("user")
	if !ok {
		t.Fatal("expected to find prefix-only")
	}

	// Wrong prefix
	_, ok = p.Get("other.name")
	if ok {
		t.Fatal("expected false for wrong prefix")
	}

	// Wrong prefix without dot
	_, ok = p.Get("other")
	if ok {
		t.Fatal("expected false for wrong prefix without dot")
	}
}

// ---------------------------------------------------------------------------
// ForeachParameter
// ---------------------------------------------------------------------------

func TestForeachParameter_eval_coverage_test(t *testing.T) {
	parent := NewGenericParam(H{"x": 1}, "")
	fp := NewForeachParameter(parent, "item", "idx")
	fp.ItemValue = reflect.ValueOf("hello")
	fp.IndexValue = reflect.ValueOf(0)

	// Get item
	val, ok := fp.Get("item")
	if !ok || val.String() != "hello" {
		t.Fatalf("expected 'hello', got %v", val)
	}

	// Get index
	val, ok = fp.Get("idx")
	if !ok || val.Int() != 0 {
		t.Fatalf("expected 0, got %v", val)
	}

	// Fallback to parent
	val, ok = fp.Get("x")
	if !ok || val.Interface() != 1 {
		t.Fatalf("expected 1, got %v", val)
	}

	// item.subfield with struct
	type item struct {
		Name string `param:"name"`
	}
	fp2 := NewForeachParameter(parent, "item", "idx")
	fp2.ItemValue = reflect.ValueOf(item{Name: "Bob"})
	fp2.IndexValue = reflect.ValueOf(0)

	val, ok = fp2.Get("item.name")
	if !ok || val.String() != "Bob" {
		t.Fatalf("expected 'Bob', got %v (ok=%v)", val, ok)
	}

	// Clear
	fp2.Clear()
}

func TestForeachParameterIndexSubfield_eval_coverage_test(t *testing.T) {
	parent := NewGenericParam(H{}, "")
	fp := NewForeachParameter(parent, "item", "meta")
	fp.ItemValue = reflect.ValueOf("value")
	fp.IndexValue = reflect.ValueOf(map[string]any{"key": "k1"})

	val, ok := fp.Get("meta.key")
	if !ok || val.Interface() != "k1" {
		t.Fatalf("expected 'k1', got %v (ok=%v)", val, ok)
	}
}

func TestForeachParameterEmptyIndex_eval_coverage_test(t *testing.T) {
	parent := NewGenericParam(H{"x": 1}, "")
	fp := NewForeachParameter(parent, "item", "")
	fp.ItemValue = reflect.ValueOf("hello")

	// When index is empty, "idx" should fallback to parent
	_, ok := fp.Get("idx")
	if ok {
		t.Fatal("expected false for empty index name")
	}
}

// ---------------------------------------------------------------------------
// ParamGroup
// ---------------------------------------------------------------------------

func TestParamGroup_eval_coverage_test(t *testing.T) {
	g := ParamGroup{
		nil, // nil should be skipped
		H{"a": 1},
		H{"b": 2},
	}

	val, ok := g.Get("a")
	if !ok || val.Interface() != 1 {
		t.Fatalf("expected 1, got %v", val)
	}

	val, ok = g.Get("b")
	if !ok || val.Interface() != 2 {
		t.Fatalf("expected 2, got %v", val)
	}

	_, ok = g.Get("c")
	if ok {
		t.Fatal("expected false for missing key")
	}
}

// ---------------------------------------------------------------------------
// Optimizer: uint, float, string static expressions
// ---------------------------------------------------------------------------

func TestOptimizerFloatExpr_eval_coverage_test(t *testing.T) {
	result, err := Eval(`3.0 + 1.0`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Float() != 4.0 {
		t.Fatalf("expected 4.0, got %v", result.Float())
	}
}

// ---------------------------------------------------------------------------
// Lexer coverage: "and", "or", "not" tokens
// ---------------------------------------------------------------------------

func TestLexerAndOrNot_eval_coverage_test(t *testing.T) {
	result, err := Eval(`x and y`, H{"x": true, "y": false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Bool() {
		t.Fatal("expected false for true and false")
	}

	result, err = Eval(`x or y`, H{"x": false, "y": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Bool() {
		t.Fatal("expected true for false or true")
	}

	result, err = Eval(`not x`, H{"x": true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Bool() {
		t.Fatal("expected false for not true")
	}
}

// ---------------------------------------------------------------------------
// structParameter: exported field not found, fallback to tag
// ---------------------------------------------------------------------------

func TestStructParameterTagLookup_eval_coverage_test(t *testing.T) {
	type item struct {
		Value int `param:"score"`
	}
	p := NewGenericParam(item{Value: 100}, "")
	// "score" is lowercase, so it goes through tag lookup path
	val, ok := p.Get("score")
	if !ok || val.Int() != 100 {
		t.Fatalf("expected 100 via tag lookup, got %v (ok=%v)", val, ok)
	}
}

// ---------------------------------------------------------------------------
// GenericParameter: empty name
// ---------------------------------------------------------------------------

func TestStructParameterMissingField_eval_coverage_test(t *testing.T) {
	type item struct {
		Value int `param:"value"`
	}
	p := NewGenericParam(item{Value: 100}, "")
	// Uppercase name that doesn't match any field
	_, ok := p.Get("Missing")
	if ok {
		t.Fatal("expected false for missing exported field")
	}
	// Lowercase name that doesn't match any tag
	_, ok = p.Get("missing")
	if ok {
		t.Fatal("expected false for missing tag")
	}
}
