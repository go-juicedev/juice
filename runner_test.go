package juice

import (
	"context"
	"errors"
	"testing"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	jsql "github.com/go-juicedev/juice/sql"
)

type resultStub struct{}

func (resultStub) LastInsertId() (int64, error) { return 0, nil }
func (resultStub) RowsAffected() (int64, error) { return 0, nil }

type statementStub struct{}

func (statementStub) ID() string { return "id" }
func (statementStub) Name() string { return "name" }
func (statementStub) Attribute(_ string) string { return "" }
func (statementStub) Build(_ driver.Translator, _ eval.Parameter) (string, []any, error) {
	return "SELECT 1", nil, nil
}
func (statementStub) Action() jsql.Action { return jsql.Select }
func (statementStub) ResultMap() (jsql.ResultMap, error) { return nil, jsql.ErrResultMapNotSet }

type sqlRowsExecutorStub struct {
	queryRows  jsql.Rows
	queryErr   error
	execResult jsql.Result
	execErr    error
	stmt       Statement
	drv        driver.Driver
}

func (s *sqlRowsExecutorStub) QueryContext(_ context.Context, _ eval.Param) (jsql.Rows, error) {
	return s.queryRows, s.queryErr
}

func (s *sqlRowsExecutorStub) ExecContext(_ context.Context, _ eval.Param) (jsql.Result, error) {
	return s.execResult, s.execErr
}

func (s *sqlRowsExecutorStub) Statement() Statement { return s.stmt }
func (s *sqlRowsExecutorStub) Driver() driver.Driver { return s.drv }

func TestErrorRunner_AllMethodsReturnSameError_runner_test(t *testing.T) {
	want := errors.New("runner failed")
	r := NewErrorRunner(want)

	if _, err := r.Select(context.Background(), nil); !errors.Is(err, want) {
		t.Fatalf("select expected %v, got %v", want, err)
	}

	if _, err := r.Insert(context.Background(), nil); !errors.Is(err, want) {
		t.Fatalf("insert expected %v, got %v", want, err)
	}

	if _, err := r.Update(context.Background(), nil); !errors.Is(err, want) {
		t.Fatalf("update expected %v, got %v", want, err)
	}

	if _, err := r.Delete(context.Background(), nil); !errors.Is(err, want) {
		t.Fatalf("delete expected %v, got %v", want, err)
	}
}

func TestGenericRunner_BindListList2_runner_test(t *testing.T) {
	rows := jsql.NewRowsBuffer([]string{"value"}, [][]any{{"a"}, {"b"}})
	r := &GenericRunner[string]{
		Runner: &ErrorRunner{error: errors.New("unused")},
	}
	r.Runner = runnerFunc(func(_ context.Context, _ eval.Param) (jsql.Rows, error) {
		return rows, nil
	})

	items, err := r.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Fatalf("unexpected list result: %#v", items)
	}

	rows2 := jsql.NewRowsBuffer([]string{"value"}, [][]any{{"x"}})
	r.Runner = runnerFunc(func(_ context.Context, _ eval.Param) (jsql.Rows, error) {
		return rows2, nil
	})

	value, err := r.Bind(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected bind error: %v", err)
	}
	if value != "x" {
		t.Fatalf("unexpected bind value: %q", value)
	}

	rows3 := jsql.NewRowsBuffer([]string{"value"}, [][]any{{"p"}, {"q"}})
	r.Runner = runnerFunc(func(_ context.Context, _ eval.Param) (jsql.Rows, error) {
		return rows3, nil
	})

	ptrItems, err := r.List2(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected list2 error: %v", err)
	}
	if len(ptrItems) != 2 || *ptrItems[0] != "p" || *ptrItems[1] != "q" {
		t.Fatalf("unexpected list2 result")
	}
}

type runnerFunc func(ctx context.Context, param eval.Param) (jsql.Rows, error)

func (f runnerFunc) Select(ctx context.Context, param eval.Param) (jsql.Rows, error) {
	return f(ctx, param)
}

func (runnerFunc) Insert(_ context.Context, _ eval.Param) (jsql.Result, error) {
	return resultStub{}, nil
}

func (runnerFunc) Update(_ context.Context, _ eval.Param) (jsql.Result, error) {
	return resultStub{}, nil
}

func (runnerFunc) Delete(_ context.Context, _ eval.Param) (jsql.Result, error) {
	return resultStub{}, nil
}

