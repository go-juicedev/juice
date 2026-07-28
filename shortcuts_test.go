package juice

import (
	"context"
	"errors"
	"testing"

	jsql "github.com/go-juicedev/juice/sql"
)

func TestShortcuts_shortcuts_test(t *testing.T) {
	executor := &sqlRowsExecutorStub{
		queryRows:  jsql.NewRowsBuffer([]string{"value"}, [][]any{{"one"}}),
		execResult: resultStub{},
		stmt:       statementStub{},
	}
	mgr := &managerStub{object: executor}
	ctx := context.Background()

	one, err := QueryContext[string](ctx, mgr, "stmt.query", H{"id": 1})
	if err != nil {
		t.Fatalf("unexpected QueryContext error: %v", err)
	}
	if one != "one" {
		t.Fatalf("unexpected QueryContext result: %q", one)
	}

	if _, err = ExecContext(ctx, mgr, "stmt.exec", H{"id": 1}); err != nil {
		t.Fatalf("unexpected ExecContext error: %v", err)
	}

	executor.queryRows = jsql.NewRowsBuffer([]string{"value"}, [][]any{{"l1"}, {"l2"}})
	items, err := QueryListContext[string](ctx, mgr, "stmt.list", nil)
	if err != nil {
		t.Fatalf("unexpected QueryListContext error: %v", err)
	}
	if len(items) != 2 || items[0] != "l1" || items[1] != "l2" {
		t.Fatalf("unexpected QueryListContext items: %#v", items)
	}

	executor.queryRows = jsql.NewRowsBuffer([]string{"value"}, [][]any{{"p1"}})
	pItems, err := QueryList2Context[string](ctx, mgr, "stmt.list2", nil)
	if err != nil {
		t.Fatalf("unexpected QueryList2Context error: %v", err)
	}
	if len(pItems) != 1 || *pItems[0] != "p1" {
		t.Fatalf("unexpected QueryList2Context items")
	}

	executor.queryRows = jsql.NewRowsBuffer([]string{"value"}, [][]any{{"i1"}, {"i2"}})
	iter, err := QueryIterContext[string](ctx, mgr, "stmt.iter", nil)
	if err != nil {
		t.Fatalf("unexpected QueryIterContext error: %v", err)
	}
	var got []string
	for item, err := range iter {
		if err != nil {
			t.Fatalf("unexpected iter item error: %v", err)
		}
		got = append(got, item)
	}
	if len(got) != 2 || got[0] != "i1" || got[1] != "i2" {
		t.Fatalf("unexpected iter items: %#v", got)
	}

	queryErr := errors.New("query failed")
	executor.queryErr = queryErr
	if _, err = QueryListContext[string](ctx, mgr, "stmt.list.err", nil); !errors.Is(err, queryErr) {
		t.Fatalf("expected query error, got %v", err)
	}
	if _, err = QueryList2Context[string](ctx, mgr, "stmt.list2.err", nil); !errors.Is(err, queryErr) {
		t.Fatalf("expected query error, got %v", err)
	}
	if _, err = QueryIterContext[string](ctx, mgr, "stmt.iter.err", nil); !errors.Is(err, queryErr) {
		t.Fatalf("expected query error, got %v", err)
	}
	executor.queryErr = nil

	execIterErr := errors.New("iter query failed")
	executor.queryErr = execIterErr
	if _, err = QueryIterContext[string](ctx, mgr, "stmt.iter.bad", nil); !errors.Is(err, execIterErr) {
		t.Fatalf("expected iterator query error, got %v", err)
	}
	executor.queryErr = nil
}
