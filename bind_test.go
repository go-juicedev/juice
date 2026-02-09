package juice

import (
	"testing"

	jsql "github.com/go-juicedev/juice/sql"
)

func TestBindWrappers_bind_test(t *testing.T) {
	rows1 := jsql.NewRowsBuffer([]string{"value"}, [][]any{{"v1"}})
	value, err := Bind[string](rows1)
	if err != nil {
		t.Fatalf("unexpected Bind error: %v", err)
	}
	if value != "v1" {
		t.Fatalf("unexpected Bind value: %q", value)
	}

	rows2 := jsql.NewRowsBuffer([]string{"value"}, [][]any{{"a"}, {"b"}})
	list, err := List[string](rows2)
	if err != nil {
		t.Fatalf("unexpected List error: %v", err)
	}
	if len(list) != 2 || list[0] != "a" || list[1] != "b" {
		t.Fatalf("unexpected List result: %#v", list)
	}

	rows3 := jsql.NewRowsBuffer([]string{"value"}, [][]any{{"x"}, {"y"}})
	list2, err := List2[string](rows3)
	if err != nil {
		t.Fatalf("unexpected List2 error: %v", err)
	}
	if len(list2) != 2 || *list2[0] != "x" || *list2[1] != "y" {
		t.Fatalf("unexpected List2 result")
	}

	rows4 := jsql.NewRowsBuffer([]string{"value"}, [][]any{{"i1"}, {"i2"}})
	iter, err := Iter[string](rows4)
	if err != nil {
		t.Fatalf("unexpected Iter error: %v", err)
	}

	var got []string
	for item, err := range iter {
		if err != nil {
			t.Fatalf("unexpected iterator item error: %v", err)
		}
		got = append(got, item)
	}

	if len(got) != 2 || got[0] != "i1" || got[1] != "i2" {
		t.Fatalf("unexpected Iter result: %#v", got)
	}

	rows5 := jsql.NewRowsBuffer([]string{"value"}, [][]any{{"r1"}})
	mapped, err := BindWithResultMap[string](rows5, jsql.SingleRowResultMap{})
	if err != nil {
		t.Fatalf("unexpected BindWithResultMap error: %v", err)
	}
	if mapped != "r1" {
		t.Fatalf("unexpected BindWithResultMap value: %q", mapped)
	}
}

