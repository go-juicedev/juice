package juice

import (
	"embed"
	"strings"
	"testing"

	jsql "github.com/go-juicedev/juice/sql"
)

//go:embed testdata/configuration
var cfg embed.FS

func TestNewXMLConfigurationWithFS_configuration_test(t *testing.T) {
	_, err := NewXMLConfigurationWithFS(cfg, "testdata/configuration/juice.xml")
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewXMLConfiguration_configuration_test(t *testing.T) {
	_, err := NewXMLConfiguration("testdata/configuration/juice.xml")
	if err != nil {
		t.Fatal(err)
	}
}

type statementIDStub struct{}

func (statementIDStub) StatementID() string {
	return "pkg.Mapper.Statement"
}

func sampleStatementFunc() {}

type emptyStatementID struct{}

func (emptyStatementID) StatementID() string { return "" }

func TestXMLConfigurationMethods_configuration_test(t *testing.T) {
	mapper := &Mapper{
		namespace: "pkg.Mapper",
		statements: map[string]*xmlSQLStatement{
			"Statement": {
				id: "Statement",
				mapper: &Mapper{
					namespace: "pkg.Mapper",
					mappers:   &Mappers{attrs: map[string]string{}},
				},
				action: jsql.Select,
			},
		},
	}

	mappers := &Mappers{mappers: nil, attrs: map[string]string{}}
	if err := mappers.setMapper("pkg.Mapper", mapper); err != nil {
		t.Fatalf("failed to set mapper: %v", err)
	}

	envs := &environments{envs: map[string]*Environment{"default": {DataSource: "dsn", Driver: "sqlite3"}}}
	settings := keyValueSettingProvider{"s": "v"}
	conf := xmlConfiguration{environments: envs, mappers: mappers, settings: settings}

	if conf.Environments() != envs {
		t.Fatalf("expected Environments passthrough")
	}
	if got := conf.Settings().Get("s"); got != "v" {
		t.Fatalf("expected settings value v, got %q", got)
	}

	if _, err := conf.GetStatement(nil); err == nil || !strings.Contains(err.Error(), "nil statement query") {
		t.Fatalf("expected nil statement error, got %v", err)
	}

	if _, err := conf.GetStatement(statementIDStub{}); err != nil {
		t.Fatalf("expected StatementID lookup success, got %v", err)
	}

	if _, err := conf.GetStatement("pkg.Mapper.Statement"); err != nil {
		t.Fatalf("expected string lookup success, got %v", err)
	}

	if _, err := conf.GetStatement(sampleStatementFunc); err == nil {
		t.Fatalf("expected function lookup fail because id mismatch")
	}

	type localStruct struct{}
	if _, err := conf.GetStatement(localStruct{}); err == nil || !strings.Contains(err.Error(), "mapper") {
		t.Fatalf("expected struct lookup fail, got %v", err)
	}

	if _, err := conf.GetStatement(123); err == nil || !strings.Contains(err.Error(), "cannot extract statement ID") {
		t.Fatalf("expected invalid type error, got %v", err)
	}

	if _, err := conf.GetStatement(emptyStatementID{}); err == nil || !strings.Contains(err.Error(), "cannot extract statement ID") {
		t.Fatalf("expected empty statement id error, got %v", err)
	}
}
