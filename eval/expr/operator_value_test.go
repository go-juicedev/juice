package expr_test

import (
	"reflect"
	"testing"

	"github.com/go-juicedev/juice/eval/expr"
)

func TestIntOperator_Addition_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(5)
	right := reflect.ValueOf(3)
	operator := expr.IntOperator{OperatorExpr: expr.Add}

	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Int() != 8 {
		t.Errorf("Expected 8, got %v", result.Int())
	}
}

func TestIntOperator_Subtraction_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(5)
	right := reflect.ValueOf(3)
	operator := expr.IntOperator{OperatorExpr: expr.Sub}

	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Int() != 2 {
		t.Errorf("Expected 2, got %v", result.Int())
	}
}

func TestIntOperator_InvalidType_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(5)
	right := reflect.ValueOf("3")
	operator := expr.IntOperator{OperatorExpr: expr.Add}

	_, err := operator.Operate(left, right)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestStringOperator_Addition_operator_value_test(t *testing.T) {
	left := reflect.ValueOf("Hello")
	right := reflect.ValueOf(" World")
	operator := expr.StringOperator{OperatorExpr: expr.Add}

	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.String() != "Hello World" {
		t.Errorf("Expected 'Hello World', got %v", result.String())
	}
}

func TestStringOperator_InvalidType_operator_value_test(t *testing.T) {
	left := reflect.ValueOf("Hello")
	right := reflect.ValueOf(3)
	operator := expr.StringOperator{OperatorExpr: expr.Add}

	_, err := operator.Operate(left, right)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestUintOperator_Addition_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(uint(5))
	right := reflect.ValueOf(uint(3))
	operator := expr.UintOperator{OperatorExpr: expr.Add}

	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Uint() != 8 {
		t.Errorf("Expected 8, got %v", result.Uint())
	}
}

func TestUintOperator_Subtraction_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(uint(5))
	right := reflect.ValueOf(uint(3))
	operator := expr.UintOperator{OperatorExpr: expr.Sub}

	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Uint() != 2 {
		t.Errorf("Expected 2, got %v", result.Uint())
	}
}

func TestUintOperator_InvalidType_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(uint(5))
	right := reflect.ValueOf("3")
	operator := expr.UintOperator{OperatorExpr: expr.Add}

	_, err := operator.Operate(left, right)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestFloatOperator_Addition_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(5.5)
	right := reflect.ValueOf(3.3)
	operator := expr.FloatOperator{OperatorExpr: expr.Add}

	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Float() != 8.8 {
		t.Errorf("Expected 8.8, got %v", result.Float())
	}
}

func TestFloatOperator_InvalidType_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(5.5)
	right := reflect.ValueOf("3.3")
	operator := expr.FloatOperator{OperatorExpr: expr.Add}

	_, err := operator.Operate(left, right)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestGenericOperator_MixedNumericPromotion_operator_value_test(t *testing.T) {
	tests := []struct {
		name     string
		left     reflect.Value
		right    reflect.Value
		operator expr.OperatorExpr
		assert   func(t *testing.T, result reflect.Value)
	}{
		{
			name:     "float plus uint promotes to float",
			left:     reflect.ValueOf(18.5),
			right:    reflect.ValueOf(uint(2)),
			operator: expr.Add,
			assert: func(t *testing.T, result reflect.Value) {
				if result.Kind() != reflect.Float64 || result.Float() != 20.5 {
					t.Fatalf("expected float64 20.5, got %v (%v)", result, result.Kind())
				}
			},
		},
		{
			name:     "int plus uint promotes to int",
			left:     reflect.ValueOf(int64(-2)),
			right:    reflect.ValueOf(uint(5)),
			operator: expr.Add,
			assert: func(t *testing.T, result reflect.Value) {
				if result.Kind() != reflect.Int64 || result.Int() != 3 {
					t.Fatalf("expected int64 3, got %v (%v)", result, result.Kind())
				}
			},
		},
		{
			name:     "complex plus float promotes to complex",
			left:     reflect.ValueOf(complex(1, 2)),
			right:    reflect.ValueOf(3.5),
			operator: expr.Add,
			assert: func(t *testing.T, result reflect.Value) {
				if result.Kind() != reflect.Complex128 || result.Complex() != complex(4.5, 2) {
					t.Fatalf("expected complex128 (4.5+2i), got %v (%v)", result, result.Kind())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := (expr.GenericOperator{OperatorExpr: tt.operator}).Operate(tt.left, tt.right)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assert(t, result)
		})
	}
}

func TestComplexOperator_Addition_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(5.5 + 3i)
	right := reflect.ValueOf(3.3 + 2i)
	operator := expr.ComplexOperator{OperatorExpr: expr.Add}

	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Complex() != 8.8+5i {
		t.Errorf("Expected 8.8 + 5i, got %v", result.Complex())
	}
}

func TestComplexOperator_InvalidType_operator_value_test(t *testing.T) {
	left := reflect.ValueOf(5.5 + 3i)
	right := reflect.ValueOf("3.3 + 2i")
	operator := expr.ComplexOperator{OperatorExpr: expr.Add}

	_, err := operator.Operate(left, right)

	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestNilEq_operator_value_test(t *testing.T) {

	left := reflect.ValueOf(new(int))
	right := reflect.ValueOf(nil)
	operator := expr.InvalidTypeOperator{OperatorExpr: expr.Eq}
	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Bool() != false {
		t.Errorf("Expected false, got %v", result.Bool())
	}
}

func TestNilNe_operator_value_test(t *testing.T) {

	left := reflect.ValueOf(new(int))
	right := reflect.ValueOf(nil)
	operator := expr.InvalidTypeOperator{OperatorExpr: expr.Ne}
	result, err := operator.Operate(left, right)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.Bool() != true {
		t.Errorf("Expected true, got %v", result.Bool())
	}
}
// ---------------------------------------------------------------------------
// UintOperator: all branches
// ---------------------------------------------------------------------------

func TestUintOperatorAllBranches_exec_coverage_test(t *testing.T) {
	left := reflect.ValueOf(uint(10))
	right := reflect.ValueOf(uint(3))

	tests := []struct {
		op     expr.OperatorExpr
		assert func(t *testing.T, rv reflect.Value)
	}{
		{expr.Add, func(t *testing.T, rv reflect.Value) { assertUint(t, rv, 13) }},
		{expr.Sub, func(t *testing.T, rv reflect.Value) { assertUint(t, rv, 7) }},
		{expr.Mul, func(t *testing.T, rv reflect.Value) { assertUint(t, rv, 30) }},
		{expr.Quo, func(t *testing.T, rv reflect.Value) { assertUint(t, rv, 3) }},
		{expr.Rem, func(t *testing.T, rv reflect.Value) { assertUint(t, rv, 1) }},
		{expr.And, func(t *testing.T, rv reflect.Value) { assertUint(t, rv, 10&3) }},
		{expr.Land, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Or, func(t *testing.T, rv reflect.Value) { assertUint(t, rv, 10|3) }},
		{expr.Lor, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Eq, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Ne, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Lt, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Le, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Gt, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Ge, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
	}

	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			rv, err := (expr.UintOperator{OperatorExpr: tt.op}).Operate(left, right)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assert(t, rv)
		})
	}

	// Unsupported operator
	if _, err := (expr.UintOperator{OperatorExpr: expr.OperatorExpr(999)}).Operate(left, right); err == nil {
		t.Fatal("expected error for unsupported uint operator")
	}
}

// ---------------------------------------------------------------------------
// FloatOperator: all branches
// ---------------------------------------------------------------------------

func TestFloatOperatorAllBranches_exec_coverage_test(t *testing.T) {
	left := reflect.ValueOf(10.0)
	right := reflect.ValueOf(3.0)

	tests := []struct {
		op     expr.OperatorExpr
		assert func(t *testing.T, rv reflect.Value)
	}{
		{expr.Add, func(t *testing.T, rv reflect.Value) { assertFloat(t, rv, 13.0) }},
		{expr.Sub, func(t *testing.T, rv reflect.Value) { assertFloat(t, rv, 7.0) }},
		{expr.Mul, func(t *testing.T, rv reflect.Value) { assertFloat(t, rv, 30.0) }},
		{expr.Quo, func(t *testing.T, rv reflect.Value) {
			if rv.Float() < 3.33 || rv.Float() > 3.34 {
				t.Fatalf("expected ~3.33, got %v", rv.Float())
			}
		}},
		{expr.Rem, func(t *testing.T, rv reflect.Value) { assertFloat(t, rv, 1.0) }},
		{expr.And, func(t *testing.T, rv reflect.Value) { assertFloat(t, rv, float64(10&3)) }},
		{expr.Land, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Or, func(t *testing.T, rv reflect.Value) { assertFloat(t, rv, float64(10|3)) }},
		{expr.Lor, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Eq, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Ne, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Lt, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Le, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Gt, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Ge, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
	}

	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			rv, err := (expr.FloatOperator{OperatorExpr: tt.op}).Operate(left, right)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assert(t, rv)
		})
	}

	// Unsupported operator
	if _, err := (expr.FloatOperator{OperatorExpr: expr.OperatorExpr(999)}).Operate(left, right); err == nil {
		t.Fatal("expected error for unsupported float operator")
	}
}

// ---------------------------------------------------------------------------
// StringOperator: all branches
// ---------------------------------------------------------------------------

func TestStringOperatorAllBranches_exec_coverage_test(t *testing.T) {
	a := reflect.ValueOf("apple")
	b := reflect.ValueOf("banana")

	tests := []struct {
		op     expr.OperatorExpr
		assert func(t *testing.T, rv reflect.Value)
	}{
		{expr.Add, func(t *testing.T, rv reflect.Value) {
			if rv.String() != "applebanana" {
				t.Fatalf("expected 'applebanana', got %v", rv.String())
			}
		}},
		{expr.Eq, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Ne, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Lt, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Le, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
		{expr.Gt, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Ge, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
	}

	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			rv, err := (expr.StringOperator{OperatorExpr: tt.op}).Operate(a, b)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assert(t, rv)
		})
	}

	// Unsupported operator
	if _, err := (expr.StringOperator{OperatorExpr: expr.Sub}).Operate(a, b); err == nil {
		t.Fatal("expected error for unsupported string operator")
	}
}

// ---------------------------------------------------------------------------
// BoolOperator: all branches
// ---------------------------------------------------------------------------

func TestBoolOperatorAllBranches_exec_coverage_test(t *testing.T) {
	T := reflect.ValueOf(true)
	F := reflect.ValueOf(false)

	tests := []struct {
		op     expr.OperatorExpr
		left   reflect.Value
		right  reflect.Value
		expect bool
	}{
		{expr.And, T, F, false},
		{expr.Land, T, T, true},
		{expr.Or, F, T, true},
		{expr.Lor, F, F, false},
		{expr.Eq, T, T, true},
		{expr.Ne, T, F, true},
	}

	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			rv, err := (expr.BoolOperator{OperatorExpr: tt.op}).Operate(tt.left, tt.right)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertBool(t, rv, tt.expect)
		})
	}
}

// ---------------------------------------------------------------------------
// ComplexOperator: all branches
// ---------------------------------------------------------------------------

func TestComplexOperatorAllBranches_exec_coverage_test(t *testing.T) {
	a := reflect.ValueOf(complex(3, 4))
	b := reflect.ValueOf(complex(1, 2))

	tests := []struct {
		op     expr.OperatorExpr
		assert func(t *testing.T, rv reflect.Value)
	}{
		{expr.Add, func(t *testing.T, rv reflect.Value) {
			if rv.Complex() != complex(4, 6) {
				t.Fatalf("expected (4+6i), got %v", rv.Complex())
			}
		}},
		{expr.Sub, func(t *testing.T, rv reflect.Value) {
			if rv.Complex() != complex(2, 2) {
				t.Fatalf("expected (2+2i), got %v", rv.Complex())
			}
		}},
		{expr.Mul, func(t *testing.T, rv reflect.Value) {
			// (3+4i)*(1+2i) = 3+6i+4i+8i² = 3+10i-8 = -5+10i
			if rv.Complex() != complex(-5, 10) {
				t.Fatalf("expected (-5+10i), got %v", rv.Complex())
			}
		}},
		{expr.Quo, func(t *testing.T, rv reflect.Value) {
			// (3+4i)/(1+2i)
			expected := complex(3, 4) / complex(1, 2)
			if rv.Complex() != expected {
				t.Fatalf("expected %v, got %v", expected, rv.Complex())
			}
		}},
		{expr.Eq, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, false) }},
		{expr.Ne, func(t *testing.T, rv reflect.Value) { assertBool(t, rv, true) }},
	}

	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			rv, err := (expr.ComplexOperator{OperatorExpr: tt.op}).Operate(a, b)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.assert(t, rv)
		})
	}

	// Unsupported operator
	if _, err := (expr.ComplexOperator{OperatorExpr: expr.Rem}).Operate(a, b); err == nil {
		t.Fatal("expected error for unsupported complex operator")
	}
}

// ---------------------------------------------------------------------------
// InvalidTypeOperator: unsupported op (not Eq or Ne)
// ---------------------------------------------------------------------------

func TestInvalidTypeOperatorUnsupported_exec_coverage_test(t *testing.T) {
	var invalid reflect.Value
	_, err := (expr.InvalidTypeOperator{OperatorExpr: expr.Add}).Operate(invalid, invalid)
	if err == nil {
		t.Fatal("expected error for unsupported op on nil values")
	}
}

func TestInvalidTypeOperatorBothValid_exec_coverage_test(t *testing.T) {
	// Both valid but entering InvalidTypeOperator (tested via GenericOperator for unknown types)
	_, err := (expr.InvalidTypeOperator{OperatorExpr: expr.Eq}).Operate(
		reflect.ValueOf(1), reflect.ValueOf("x"),
	)
	if err == nil {
		t.Fatal("expected error for mismatched valid types")
	}
}

// ---------------------------------------------------------------------------
// GenericOperator: numeric promotion paths
// ---------------------------------------------------------------------------

func TestGenericOperatorUintOnly_exec_coverage_test(t *testing.T) {
	rv, err := (expr.GenericOperator{OperatorExpr: expr.Add}).Operate(
		reflect.ValueOf(uint(5)), reflect.ValueOf(uint(3)),
	)
	if err != nil || rv.Uint() != 8 {
		t.Fatalf("expected uint 8, got err=%v rv=%v", err, rv)
	}
}

func TestGenericOperatorIntAndUint_exec_coverage_test(t *testing.T) {
	rv, err := (expr.GenericOperator{OperatorExpr: expr.Add}).Operate(
		reflect.ValueOf(int64(5)), reflect.ValueOf(uint(3)),
	)
	if err != nil || rv.Int() != 8 {
		t.Fatalf("expected int64 8, got err=%v rv=%v", err, rv)
	}
}

func TestGenericOperatorComplexAndInt_exec_coverage_test(t *testing.T) {
	rv, err := (expr.GenericOperator{OperatorExpr: expr.Add}).Operate(
		reflect.ValueOf(complex(1, 2)), reflect.ValueOf(int64(3)),
	)
	if err != nil || rv.Complex() != complex(4, 2) {
		t.Fatalf("expected (4+2i), got err=%v rv=%v", err, rv)
	}
}

func TestGenericOperatorComplexAndUint_exec_coverage_test(t *testing.T) {
	rv, err := (expr.GenericOperator{OperatorExpr: expr.Add}).Operate(
		reflect.ValueOf(complex(1, 2)), reflect.ValueOf(uint(3)),
	)
	if err != nil || rv.Complex() != complex(4, 2) {
		t.Fatalf("expected (4+2i), got err=%v rv=%v", err, rv)
	}
}

func TestGenericOperatorFloatAndUint_exec_coverage_test(t *testing.T) {
	rv, err := (expr.GenericOperator{OperatorExpr: expr.Add}).Operate(
		reflect.ValueOf(1.5), reflect.ValueOf(uint(2)),
	)
	if err != nil || rv.Float() != 3.5 {
		t.Fatalf("expected 3.5, got err=%v rv=%v", err, rv)
	}
}

// ---------------------------------------------------------------------------
// allOf / anyOf edge cases (empty values)
// ---------------------------------------------------------------------------

func TestAllOfEmpty_exec_coverage_test(t *testing.T) {
	op := expr.GenericOperator{OperatorExpr: expr.Add}
	// This tests the internal allOf with no matching predicate
	_, err := op.Operate(reflect.ValueOf(struct{}{}), reflect.ValueOf(struct{}{}))
	if err == nil {
		t.Fatal("expected error for unsupported struct type")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type errForTest string

func (e errForTest) Error() string { return string(e) }

func assertBool(t *testing.T, rv reflect.Value, want bool) {
	t.Helper()
	if rv.Bool() != want {
		t.Fatalf("expected %v, got %v", want, rv.Bool())
	}
}

func assertUint(t *testing.T, rv reflect.Value, want uint64) {
	t.Helper()
	if rv.Uint() != want {
		t.Fatalf("expected %d, got %v", want, rv.Uint())
	}
}

func assertFloat(t *testing.T, rv reflect.Value, want float64) {
	t.Helper()
	if rv.Float() != want {
		t.Fatalf("expected %v, got %v", want, rv.Float())
	}
}
