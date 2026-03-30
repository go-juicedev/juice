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
