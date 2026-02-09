package ctxreducer_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/internal/ctxreducer"
	"github.com/go-juicedev/juice/session"
)

type dummySession struct{}

func (dummySession) QueryContext(context.Context, string, ...any) (*sql.Rows, error) { return nil, nil }
func (dummySession) ExecContext(context.Context, string, ...any) (sql.Result, error) { return nil, nil }
func (dummySession) PrepareContext(context.Context, string) (*sql.Stmt, error)        { return nil, nil }

func TestContextReducerFunc_Reduce_reducer_test(t *testing.T) {
	type key struct{}

	reducer := ctxreducer.ContextReducerFunc(func(ctx context.Context) context.Context {
		return context.WithValue(ctx, key{}, 123)
	})

	ctx := reducer.Reduce(context.Background())
	if got := ctx.Value(key{}); got != 123 {
		t.Fatalf("unexpected value in context: got=%v want=%v", got, 123)
	}
}

func TestContextReducerGroup_ReduceOrder_reducer_test(t *testing.T) {
	var calls []int
	reducer1 := ctxreducer.ContextReducerFunc(func(ctx context.Context) context.Context {
		calls = append(calls, 1)
		return ctx
	})
	reducer2 := ctxreducer.ContextReducerFunc(func(ctx context.Context) context.Context {
		calls = append(calls, 2)
		return ctx
	})

	group := ctxreducer.ContextReducerGroup{reducer1, reducer2}
	_ = group.Reduce(context.Background())

	if !reflect.DeepEqual(calls, []int{1, 2}) {
		t.Fatalf("unexpected reducer call order: got=%v want=%v", calls, []int{1, 2})
	}
}

func TestContextReducerGroupAlias_Reduce_reducer_test(t *testing.T) {
	var group ctxreducer.G
	ctx := group.Reduce(context.Background())
	if ctx != context.Background() {
		t.Fatalf("unexpected reduced context: got=%v want=%v", ctx, context.Background())
	}
}

func TestNewParamContextReducer_Reduce_reducer_test(t *testing.T) {
	want := "hello"
	reducer := ctxreducer.NewParamContextReducer(want)

	ctx := reducer.Reduce(context.Background())
	got := eval.ParamFromContext(ctx)
	if got != want {
		t.Fatalf("unexpected param from context: got=%v want=%v", got, want)
	}
}

func TestNewSessionContextReducer_Reduce_reducer_test(t *testing.T) {
	want := dummySession{}
	reducer := ctxreducer.NewSessionContextReducer(want)

	ctx := reducer.Reduce(context.Background())
	got, err := session.FromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected session: got=%T want=%T", got, want)
	}
}

