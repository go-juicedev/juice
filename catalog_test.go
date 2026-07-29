package juice

import (
	"errors"
	"testing"
)

func TestStatementCatalogExactLookup(t *testing.T) {
	catalog := newStatementCatalog()
	statements := []*mappedStatement{
		{id: "example.shared.UserMapper.Find"},
		{id: "example.shared.UserMapper.List"},
		{id: "example.shared.OrderMapper.Find"},
	}

	for _, statement := range statements {
		if err := catalog.add(statement); err != nil {
			t.Fatalf("add statement %q: %v", statement.ID(), err)
		}
	}
	for _, statement := range statements {
		if got, err := catalog.Statement(statement.ID()); err != nil || got != statement {
			t.Fatalf("Statement(%q) = (%v, %v)", statement.ID(), got, err)
		}
	}
	if _, err := catalog.Statement("example.shared.UserMapper"); !errors.Is(err, ErrNoStatementFound) {
		t.Fatalf("partial lookup error = %v, want %v", err, ErrNoStatementFound)
	}
	if err := catalog.add(statements[0]); err == nil {
		t.Fatal("duplicate statement id was accepted")
	}
}

func TestNilStatementCatalogReturnsNotFound(t *testing.T) {
	var catalog *statementCatalog
	if _, err := catalog.Statement("example.Mapper.One"); !errors.Is(err, ErrNoStatementFound) {
		t.Fatalf("nil catalog error = %v, want %v", err, ErrNoStatementFound)
	}
}
