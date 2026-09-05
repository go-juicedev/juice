package juice

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	xmlparser "github.com/go-juicedev/juice/parser/xml"
	jsql "github.com/go-juicedev/juice/sql"
)

func TestNewXMLConfigurationWithFS_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"configuration/juice.xml": {
			Data: []byte(`
<configuration>
	<environments default="prod">
		<environment id="prod">
			<dataSource>fake</dataSource>
			<driver>fake</driver>
		</environment>
	</environments>
	<mappers pattern="mappers/*.xml"/>
</configuration>`),
		},
		"configuration/mappers/mapper.xml": {
			Data: []byte(`
<mapper namespace="pkg.Mapper">
	<select id="Select">SELECT 1</select>
</mapper>`),
		},
	}

	configuration, err := NewXMLConfigurationWithFS(fsys, "configuration/juice.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := configuration.Backend().(xmlparser.Backend); !ok {
		t.Fatalf("expected XML backend, got %T", configuration.Backend())
	}
}

func TestNewXMLConfiguration_configuration_test(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "juice.xml")
	err := os.WriteFile(filename, []byte(`
<configuration>
	<environments default="prod">
		<environment id="prod">
			<dataSource>fake</dataSource>
			<driver>fake</driver>
		</environment>
	</environments>
</configuration>`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewXMLConfiguration(filename)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewXMLConfiguration_emptyPath_configuration_test(t *testing.T) {
	if _, err := NewXMLConfiguration(""); err == nil || !strings.Contains(err.Error(), "configuration path is required") {
		t.Fatalf("expected empty path error, got %v", err)
	}
}

func TestNewXMLConfigurationWithFS_emptyPath_configuration_test(t *testing.T) {
	if _, err := NewXMLConfigurationWithFS(fstest.MapFS{}, ""); err == nil || !strings.Contains(err.Error(), "configuration path is required") {
		t.Fatalf("expected empty path error, got %v", err)
	}
}

func TestNewXMLConfigurationWithFSMissingEnvironments_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<mappers>
		<mapper namespace="pkg.Mapper">
			<select id="Select">SELECT 1</select>
		</mapper>
	</mappers>
</configuration>`),
		},
	}

	_, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if !errors.Is(err, errConfigurationEnvironmentsRequired) {
		t.Fatalf("expected missing environments error, got %v", err)
	}
}

func TestNewXMLConfigurationWithFSMissingMappers_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<environments default="prod">
		<environment id="prod">
			<dataSource>sqlite.db</dataSource>
			<driver>sqlite3</driver>
		</environment>
	</environments>
</configuration>`),
		},
	}

	if _, err := NewXMLConfigurationWithFS(fsys, "juice.xml"); err != nil {
		t.Fatalf("expected missing mappers to be allowed, got %v", err)
	}
}

func TestNewXMLConfigurationWithFSEmptyMappers_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<environments default="prod">
		<environment id="prod">
			<dataSource>sqlite.db</dataSource>
			<driver>sqlite3</driver>
		</environment>
	</environments>
	<mappers />
</configuration>`),
		},
	}

	if _, err := NewXMLConfigurationWithFS(fsys, "juice.xml"); err != nil {
		t.Fatalf("expected empty mappers to be allowed, got %v", err)
	}
}

func TestNewXMLConfigurationWithFSResourceMapperRequiresMapperRoot_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<environments default="prod">
		<environment id="prod">
			<dataSource>sqlite.db</dataSource>
			<driver>sqlite3</driver>
		</environment>
	</environments>
	<mappers>
		<mapper resource="not-a-mapper.xml"/>
	</mappers>
</configuration>`),
		},
		"not-a-mapper.xml": {
			Data: []byte(`<notMapper/>`),
		},
	}

	_, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if !errors.Is(err, errMapperRootElementNotFound) {
		t.Fatalf("expected mapper root error, got %v", err)
	}
}

func TestNewXMLConfigurationWithFSPatternMapperRequiresMapperRoot_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<environments default="prod">
		<environment id="prod">
			<dataSource>sqlite.db</dataSource>
			<driver>sqlite3</driver>
		</environment>
	</environments>
	<mappers pattern="mappers/*.xml" />
</configuration>`),
		},
		"mappers/not-a-mapper.xml": {
			Data: []byte(`<notMapper/>`),
		},
	}

	_, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if !errors.Is(err, errMapperRootElementNotFound) {
		t.Fatalf("expected mapper root error, got %v", err)
	}
}

func TestNewXMLConfigurationWithFSUnknownDefaultEnvironment_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<environments default="prod">
		<environment id="dev">
			<dataSource>sqlite.db</dataSource>
			<driver>sqlite3</driver>
		</environment>
	</environments>
	<mappers>
		<mapper namespace="pkg.Mapper">
			<select id="Select">SELECT 1</select>
		</mapper>
	</mappers>
</configuration>`),
		},
	}

	_, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if !errors.Is(err, errConfigurationDefaultEnvironmentUnknown) {
		t.Fatalf("expected unknown default environment error, got %v", err)
	}
}

func TestNewXMLConfigurationWithFSUnknownEnvValueProvider_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<environments default="prod">
		<environment id="prod" provider="missing">
			<dataSource>sqlite.db</dataSource>
			<driver>sqlite3</driver>
		</environment>
	</environments>
	<mappers>
		<mapper namespace="pkg.Mapper">
			<select id="Select">SELECT 1</select>
		</mapper>
	</mappers>
</configuration>`),
		},
	}

	_, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if err == nil || !strings.Contains(err.Error(), "environment value provider not found") {
		t.Fatalf("expected unknown env value provider error, got %v", err)
	}
}

func TestNewXMLConfigurationWithFSEmptyEnvValueProvider_configuration_test(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<configuration>
	<environments default="prod">
		<environment id="prod" provider="">
			<dataSource>sqlite.db</dataSource>
			<driver>sqlite3</driver>
		</environment>
	</environments>
	<mappers>
		<mapper namespace="pkg.Mapper">
			<select id="Select">SELECT 1</select>
		</mapper>
	</mappers>
</configuration>`),
		},
	}

	if _, err := NewXMLConfigurationWithFS(fsys, "juice.xml"); err != nil {
		t.Fatalf("expected empty env value provider to use passthrough provider, got %v", err)
	}
}

type statementIDStub struct{}

func (statementIDStub) StatementID() string {
	return "pkg.Mapper.Statement"
}

func sampleStatementFunc() {}

type emptyStatementID struct{}

func (emptyStatementID) StatementID() string { return "" }

func TestConfigurationMethods_configuration_test(t *testing.T) {
	statement := &mappedStatement{
		id:     "pkg.Mapper.Statement",
		action: jsql.Select,
	}
	catalog := newStatementCatalog()
	if err := catalog.add(statement); err != nil {
		t.Fatalf("failed to add statement: %v", err)
	}

	settings := keyValueSettingProvider{"s": "v"}
	conf := compiledConfig{
		catalog: catalog,
		runtime: &RuntimeConfig{
			defaultSource: "default",
			sources:       map[string]Source{"default": {DSN: "dsn", Driver: "sqlite3"}},
			settings:      settings,
		},
	}

	if got := conf.Settings().Get("s"); got != "v" {
		t.Fatalf("expected settings value v, got %q", got)
	}

}
