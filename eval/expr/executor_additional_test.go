package expr_test

import (
	"errors"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/go-juicedev/juice/eval/expr"
)

func TestOperatorExprString_executor_additional_test(t *testing.T) {
	table := []struct {
		op   expr.OperatorExpr
		want string
	}{
		{expr.Add, "+"},
		{expr.Sub, "-"},
		{expr.Mul, "*"},
		{expr.Quo, "/"},
		{expr.Rem, "%"},
		{expr.And, "&"},
		{expr.Land, "&&"},
		{expr.Or, "|"},
		{expr.Lor, "||"},
		{expr.Eq, "=="},
		{expr.Ne, "!="},
		{expr.Lt, "<"},
		{expr.Le, "<="},
		{expr.Gt, ">"},
		{expr.Ge, ">="},
		{expr.OperatorExpr(999), ""},
	}

	for _, tt := range table {
		if got := tt.op.String(); got != tt.want {
			t.Fatalf("operator %v expected %q, got %q", tt.op, tt.want, got)
		}
	}
}

func TestOperationErrorMessage_executor_additional_test(t *testing.T) {
	err := expr.NewOperationError(reflect.ValueOf(1), reflect.ValueOf("x"), "+")
	if err == nil {
		t.Fatalf("expected error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "invalid operation +") || !strings.Contains(msg, "int") || !strings.Contains(msg, "string") {
		t.Fatalf("unexpected operation error: %q", msg)
	}
}

func TestIntOperatorMoreBranches_executor_additional_test(t *testing.T) {
	left := reflect.ValueOf(6)
	right := reflect.ValueOf(4)

	table := []struct {
		op     expr.OperatorExpr
		assert func(t *testing.T, rv reflect.Value)
	}{
		{expr.Mul, func(t *testing.T, rv reflect.Value) { if rv.Int() != 24 { t.Fatalf("expected 24") } }},
		{expr.Quo, func(t *testing.T, rv reflect.Value) { if rv.Int() != 1 { t.Fatalf("expected 1") } }},
		{expr.Rem, func(t *testing.T, rv reflect.Value) { if rv.Int() != 2 { t.Fatalf("expected 2") } }},
		{expr.And, func(t *testing.T, rv reflect.Value) { if rv.Int() != 4 { t.Fatalf("expected 4") } }},
		{expr.Land, func(t *testing.T, rv reflect.Value) { if !rv.Bool() { t.Fatalf("expected true") } }},
		{expr.Or, func(t *testing.T, rv reflect.Value) { if rv.Int() != 6 { t.Fatalf("expected 6") } }},
		{expr.Lor, func(t *testing.T, rv reflect.Value) { if !rv.Bool() { t.Fatalf("expected true") } }},
		{expr.Eq, func(t *testing.T, rv reflect.Value) { if rv.Bool() { t.Fatalf("expected false") } }},
		{expr.Ne, func(t *testing.T, rv reflect.Value) { if !rv.Bool() { t.Fatalf("expected true") } }},
		{expr.Lt, func(t *testing.T, rv reflect.Value) { if rv.Bool() { t.Fatalf("expected false") } }},
		{expr.Le, func(t *testing.T, rv reflect.Value) { if rv.Bool() { t.Fatalf("expected false") } }},
		{expr.Gt, func(t *testing.T, rv reflect.Value) { if !rv.Bool() { t.Fatalf("expected true") } }},
		{expr.Ge, func(t *testing.T, rv reflect.Value) { if !rv.Bool() { t.Fatalf("expected true") } }},
	}

	for _, tt := range table {
		rv, err := (expr.IntOperator{OperatorExpr: tt.op}).Operate(left, right)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.op.String(), err)
		}
		tt.assert(t, rv)
	}

	if _, err := (expr.IntOperator{OperatorExpr: expr.OperatorExpr(999)}).Operate(left, right); err == nil {
		t.Fatalf("expected error for unsupported operator")
	}
}

func TestOtherOperatorsMoreBranches_executor_additional_test(t *testing.T) {
	if rv, err := (expr.UintOperator{OperatorExpr: expr.Mul}).Operate(reflect.ValueOf(uint(6)), reflect.ValueOf(uint(4))); err != nil || rv.Uint() != 24 {
		t.Fatalf("unexpected uint mul result err=%v rv=%v", err, rv)
	}

	if rv, err := (expr.FloatOperator{OperatorExpr: expr.Rem}).Operate(reflect.ValueOf(7.0), reflect.ValueOf(3.0)); err != nil || rv.Float() != 1 {
		t.Fatalf("unexpected float rem result err=%v rv=%v", err, rv)
	}

	if rv, err := (expr.StringOperator{OperatorExpr: expr.Eq}).Operate(reflect.ValueOf("a"), reflect.ValueOf("a")); err != nil || !rv.Bool() {
		t.Fatalf("unexpected string eq result err=%v rv=%v", err, rv)
	}

	if rv, err := (expr.BoolOperator{OperatorExpr: expr.Or}).Operate(reflect.ValueOf(true), reflect.ValueOf(false)); err != nil || !rv.Bool() {
		t.Fatalf("unexpected bool or result err=%v rv=%v", err, rv)
	}

	if rv, err := (expr.ComplexOperator{OperatorExpr: expr.Ne}).Operate(reflect.ValueOf(complex(1, 2)), reflect.ValueOf(complex(2, 2))); err != nil || !rv.Bool() {
		t.Fatalf("unexpected complex ne result err=%v rv=%v", err, rv)
	}

	if _, err := (expr.BoolOperator{OperatorExpr: expr.Add}).Operate(reflect.ValueOf(true), reflect.ValueOf(false)); err == nil {
		t.Fatalf("expected unsupported bool operator error")
	}
}

func TestGenericAndInvalidTypeOperators_executor_additional_test(t *testing.T) {
	var invalid reflect.Value

	rv, err := (expr.InvalidTypeOperator{OperatorExpr: expr.Eq}).Operate(invalid, invalid)
	if err != nil {
		t.Fatalf("unexpected nil eq error: %v", err)
	}
	if !rv.Bool() {
		t.Fatalf("expected nil eq to be true")
	}

	rv, err = (expr.InvalidTypeOperator{OperatorExpr: expr.Ne}).Operate(invalid, invalid)
	if err != nil {
		t.Fatalf("unexpected nil ne error: %v", err)
	}
	if rv.Bool() {
		t.Fatalf("expected nil ne to be false")
	}

	if _, err = (expr.GenericOperator{OperatorExpr: expr.Add}).Operate(reflect.ValueOf(1), reflect.ValueOf("x")); err == nil {
		t.Fatalf("expected mixed-type operation error")
	}

	rv, err = (expr.GenericOperator{OperatorExpr: expr.Eq}).Operate(reflect.ValueOf(true), reflect.ValueOf(true))
	if err != nil || !rv.Bool() {
		t.Fatalf("unexpected bool eq result err=%v rv=%v", err, rv)
	}
}

func TestOperatorExecutorAndExpressionExecutors_executor_additional_test(t *testing.T) {
	want := errors.New("x failed")
	op := expr.OperatorExecutor{Operator: expr.GenericOperator{OperatorExpr: expr.Add}}

	if _, err := op.Exec(func() (reflect.Value, error) { return reflect.Value{}, want }, nil); !errors.Is(err, want) {
		t.Fatalf("expected x error, got %v", err)
	}

	if _, err := op.Exec(
		func() (reflect.Value, error) { return reflect.ValueOf(1), nil },
		func() (reflect.Value, error) { return reflect.Value{}, want },
	); !errors.Is(err, want) {
		t.Fatalf("expected y error, got %v", err)
	}

	rv, err := op.Exec(
		func() (reflect.Value, error) { return reflect.ValueOf(1), nil },
		func() (reflect.Value, error) { return reflect.ValueOf(2), nil },
	)
	if err != nil || rv.Int() != 3 {
		t.Fatalf("unexpected operator exec result err=%v rv=%v", err, rv)
	}

	yCalled := false
	rv, err = (expr.LANDExprExecutor{}).Exec(
		func() (reflect.Value, error) { return reflect.ValueOf(false), nil },
		func() (reflect.Value, error) { yCalled = true; return reflect.ValueOf(true), nil },
	)
	if err != nil || rv.Bool() || yCalled {
		t.Fatalf("expected short-circuit false, err=%v rv=%v yCalled=%v", err, rv, yCalled)
	}

	rv, err = (expr.LORExprExecutor{}).Exec(
		func() (reflect.Value, error) { return reflect.ValueOf(true), nil },
		func() (reflect.Value, error) { yCalled = true; return reflect.ValueOf(false), nil },
	)
	if err != nil || !rv.Bool() {
		t.Fatalf("unexpected lor result err=%v rv=%v", err, rv)
	}

	if _, err = (expr.LANDExprExecutor{}).Exec(
		func() (reflect.Value, error) { return reflect.ValueOf(1), nil },
		func() (reflect.Value, error) { return reflect.ValueOf(true), nil },
	); err == nil {
		t.Fatalf("expected left type error")
	}

	if _, err = (expr.NOTExprExecutor{}).Exec(nil, func() (reflect.Value, error) { return reflect.ValueOf(true), nil }); err != nil {
		t.Fatalf("unexpected not error: %v", err)
	}

	if _, err = (expr.NOTExprExecutor{}).Exec(nil, func() (reflect.Value, error) { return reflect.ValueOf(1), nil }); err == nil {
		t.Fatalf("expected not type error")
	}

	if _, err = (expr.ANDExprExecutor{}).Exec(
		func() (reflect.Value, error) { return reflect.ValueOf(true), nil },
		func() (reflect.Value, error) { return reflect.ValueOf(true), nil },
	); err != nil {
		t.Fatalf("unexpected and error: %v", err)
	}

	if _, err = (expr.ORExprExecutor{}).Exec(
		func() (reflect.Value, error) { return reflect.ValueOf(false), nil },
		func() (reflect.Value, error) { return reflect.ValueOf(true), nil },
	); err != nil {
		t.Fatalf("unexpected or error: %v", err)
	}

	if rv, err = (expr.LPARENExprExecutor{}).Exec(nil, func() (reflect.Value, error) { return reflect.ValueOf(7), nil }); err != nil || rv.Int() != 7 {
		t.Fatalf("unexpected lparen result err=%v rv=%v", err, rv)
	}

	if rv, err = (expr.RPARENExprExecutor{}).Exec(func() (reflect.Value, error) { return reflect.ValueOf(8), nil }, nil); err != nil || rv.Int() != 8 {
		t.Fatalf("unexpected rparen result err=%v rv=%v", err, rv)
	}

	if rv, err = (expr.COMMENTExprExecutor{}).Exec(nil, nil); err != nil || !rv.Bool() {
		t.Fatalf("unexpected comment result err=%v rv=%v", err, rv)
	}
}

func TestFromToken_executor_additional_test(t *testing.T) {
	supported := []token.Token{
		token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ,
		token.LAND, token.LOR, token.ADD, token.SUB, token.MUL, token.QUO,
		token.REM, token.LPAREN, token.RPAREN, token.COMMENT, token.NOT,
		token.AND, token.OR,
	}

	for _, tok := range supported {
		exe, err := expr.FromToken(tok)
		if err != nil {
			t.Fatalf("unexpected error for token %v: %v", tok, err)
		}
		if exe == nil {
			t.Fatalf("expected non-nil executor for token %v", tok)
		}
	}

	_, err := expr.FromToken(token.ILLEGAL)
	if !errors.Is(err, expr.ErrUnsupportedBinaryExpr) {
		t.Fatalf("expected ErrUnsupportedBinaryExpr, got %v", err)
	}
}
