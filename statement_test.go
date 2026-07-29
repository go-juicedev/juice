package juice

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
	xmlparser "github.com/go-juicedev/juice/parser/xml"
	jsql "github.com/go-juicedev/juice/sql"
)

func TestMappedStatement_MetadataAndBuild_statement_test(t *testing.T) {
	stmt := &mappedStatement{
		action: jsql.Select,
		Nodes: node.Group{
			xmlparser.NewTextNode("SELECT 1"),
		},
		attrs: map[string]string{"timeout": "3s"},
		id:    "app.user.SelectOne",
	}
	stmt.setAttribute("local", "enabled")

	if got := stmt.ID(); got != "app.user.SelectOne" {
		t.Fatalf("unexpected statement id: %q", got)
	}

	if got := stmt.Attribute("local"); got != "enabled" {
		t.Fatalf("expected local attr from statement, got %q", got)
	}

	if got := stmt.Attribute("timeout"); got != "3s" {
		t.Fatalf("expected fallback attr from mapper, got %q", got)
	}

	if got := stmt.Attribute("missing"); got != "" {
		t.Fatalf("expected empty attr for missing key, got %q", got)
	}

	if got := stmt.Action(); got != jsql.Select {
		t.Fatalf("expected select action, got %q", got)
	}

	if _, err := stmt.ResultMap(); !errors.Is(err, jsql.ErrResultMapNotSet) {
		t.Fatalf("expected ErrResultMapNotSet, got %v", err)
	}

	query, args, err := stmt.Build(driver.TranslateFunc(func(_ string) string { return "?" }), eval.H{})
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if query != "SELECT 1" {
		t.Fatalf("unexpected query: %q", query)
	}

	if len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}

func TestMappedStatement_BuildEmptyQuery_statement_test(t *testing.T) {
	stmt := &mappedStatement{
		id: "ns.Empty",
	}

	_, _, err := stmt.Build(driver.TranslateFunc(func(_ string) string { return "?" }), eval.H{})
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestRawSQLStatement_MetadataAndBuild_statement_test(t *testing.T) {
	stmt := NewRawSQLStatement(xmlparser.Backend{}, "SELECT * FROM ${table} WHERE id = #{id}", jsql.Select)
	stmt.WithAttribute("cache", "true")

	if got := stmt.Attribute("cache"); got != "true" {
		t.Fatalf("expected cache attr true, got %q", got)
	}

	if got := stmt.Attribute("missing"); got != "" {
		t.Fatalf("expected missing attr empty, got %q", got)
	}

	id := stmt.ID()
	if !strings.HasPrefix(id.String(), "id:") {
		t.Fatalf("expected id prefix id:, got %q", id)
	}

	if got := stmt.Action(); got != jsql.Select {
		t.Fatalf("expected select action, got %q", got)
	}

	if _, err := stmt.ResultMap(); !errors.Is(err, jsql.ErrResultMapNotSet) {
		t.Fatalf("expected ErrResultMapNotSet, got %v", err)
	}

	query, args, err := stmt.Build(driver.TranslateFunc(func(_ string) string { return "?" }), eval.H{
		"table": "users",
		"id":    7,
	})
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if query != "SELECT * FROM users WHERE id = ?" {
		t.Fatalf("unexpected query: %q", query)
	}

	if len(args) != 1 || args[0] != 7 {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestRawSQLStatement_BuildEmptyQuery_statement_test(t *testing.T) {
	stmt := NewRawSQLStatement(xmlparser.Backend{}, "", jsql.Select)
	_, _, err := stmt.Build(driver.TranslateFunc(func(_ string) string { return "?" }), eval.H{})
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}
