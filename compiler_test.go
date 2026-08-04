package juice

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	configparser "github.com/go-juicedev/juice/parser"
	xmlparser "github.com/go-juicedev/juice/parser/xml"
)

func TestCompileBuildsRuntimeConfigWithInjectedEnvironmentProvider(t *testing.T) {
	document := &configparser.Document{
		Settings: map[string]string{"debug": "false"},
		Environments: configparser.Environments{
			Default: "primary",
			Items: []configparser.Environment{
				{
					ID:                  "primary",
					Driver:              "driver-key",
					DataSource:          "dsn-key",
					MaxIdleConns:        "max-idle-key",
					MaxOpenConns:        "max-open-key",
					ConnMaxLifetime:     "max-lifetime-key",
					ConnMaxIdleLifetime: "max-idle-lifetime-key",
					Attributes:          map[string]string{"provider": "injected"},
				},
			},
		},
	}
	values := map[string]string{
		"driver-key":            "postgres",
		"dsn-key":               "postgres://original",
		"max-idle-key":          "3",
		"max-open-key":          "9",
		"max-lifetime-key":      "60",
		"max-idle-lifetime-key": "30",
	}
	lookupCalls := 0
	lookup := func(name string) (EnvValueProvider, bool) {
		lookupCalls++
		if name != "injected" {
			return nil, false
		}
		return EnvValueProviderFunc(func(key string) (string, error) {
			return values[key], nil
		}), true
	}

	compiled, err := Compile(document, CompileOptions{
		Backend:                xmlparser.Backend{},
		EnvValueProviderLookup: lookup,
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if lookupCalls != 1 {
		t.Fatalf("provider lookup calls = %d, want 1", lookupCalls)
	}

	document.Settings["debug"] = "true"
	document.Environments.Items[0].Driver = "mutated"

	source, exists := compiled.Source("primary")
	if !exists {
		t.Fatal("compiled primary source not found")
	}
	if source.Driver != "postgres" || source.DSN != "postgres://original" ||
		source.MaxIdleConns != 3 || source.MaxOpenConns != 9 ||
		source.ConnMaxLifetime != time.Minute || source.ConnMaxIdleTime != 30*time.Second {
		t.Fatalf("compiled source changed after source mutation: %#v", source)
	}
	if compiled.Settings().Get("debug") != "false" {
		t.Fatalf("compiled setting changed after document mutation: %q", compiled.Settings().Get("debug"))
	}
}

func TestCompileRejectsNilDocument(t *testing.T) {
	if _, err := Compile(nil, CompileOptions{}); !errors.Is(err, errConfigurationRequired) {
		t.Fatalf("Compile(nil) error = %v, want %v", err, errConfigurationRequired)
	}
}

func TestCompileRequiresBackend(t *testing.T) {
	document := &configparser.Document{
		Environments: configparser.Environments{
			Default: "primary",
			Items:   []configparser.Environment{{ID: "primary"}},
		},
	}
	if _, err := Compile(document, CompileOptions{}); !errors.Is(err, errConfigurationBackendRequired) {
		t.Fatalf("Compile() error = %v, want %v", err, errConfigurationBackendRequired)
	}
}

func TestCompileIgnoreEnvironmentSkipsEnvironmentResolution(t *testing.T) {
	document := &configparser.Document{
		Environments: configparser.Environments{
			Default: "primary",
			Items: []configparser.Environment{
				{ID: "primary", Driver: "driver", Attributes: map[string]string{"provider": "missing"}},
			},
		},
	}
	lookupCalled := false
	compiled, err := Compile(document, CompileOptions{
		Backend:           xmlparser.Backend{},
		IgnoreEnvironment: true,
		EnvValueProviderLookup: func(string) (EnvValueProvider, bool) {
			lookupCalled = true
			return nil, false
		},
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if lookupCalled {
		t.Fatal("environment provider was resolved while environments were ignored")
	}
	if compiled.DefaultSource() != "" {
		t.Fatalf("default source = %q, want empty", compiled.DefaultSource())
	}
}

func TestCompileValidatesMapperStatements(t *testing.T) {
	tests := []struct {
		name   string
		mapper configparser.Mapper
		want   string
	}{
		{name: "missing namespace", mapper: configparser.Mapper{}, want: "mapper namespace is required"},
		{
			name: "missing statement id",
			mapper: configparser.Mapper{
				Namespace:  "example.Mapper",
				Statements: []configparser.Statement{{Action: configparser.Select, Node: xmlparser.NewTextNode("select 1")}},
			},
			want: "statement id is required",
		},
		{
			name: "invalid action",
			mapper: configparser.Mapper{
				Namespace:  "example.Mapper",
				Statements: []configparser.Statement{{ID: "Find", Action: "merge", Node: xmlparser.NewTextNode("select 1")}},
			},
			want: "invalid action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			document := &configparser.Document{Mappers: []configparser.Mapper{tt.mapper}}
			_, err := Compile(document, CompileOptions{
				Backend:           xmlparser.Backend{},
				IgnoreEnvironment: true,
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Compile() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestXMLConfigurationCompilesCanonicalStatementCatalog(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`
<configuration>
    <environments default="prod">
        <environment id="prod"><driver>mysql</driver><dataSource>dsn</dataSource></environment>
    </environments>
	<mappers>
		<mapper namespace="example.Mapper">
			<select id="One" timeout="1000">select 1</select>
        </mapper>
    </mappers>
</configuration>`)},
	}

	configuration, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if err != nil {
		t.Fatal(err)
	}

	id := StatementID("example.Mapper.One")
	statement, err := configuration.Statement(id)
	if err != nil {
		t.Fatal(err)
	}
	if statement.ID() != id {
		t.Fatalf("statement id = %q, want %q", statement.ID(), id)
	}
	if statement.Attribute("timeout") != "1000" {
		t.Fatalf("statement timeout = %q, want 1000", statement.Attribute("timeout"))
	}
}

func TestXMLConfigurationMergesMapperDocumentsByNamespace(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`
<configuration>
    <environments default="prod">
        <environment id="prod"><driver>mysql</driver><dataSource>dsn</dataSource></environment>
    </environments>
    <mappers>
        <mapper namespace="example.Mapper"><select id="One">select 1</select></mapper>
        <mapper namespace="example.Mapper"><select id="Two">select 2</select></mapper>
    </mappers>
</configuration>`)},
	}

	configuration, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []StatementID{"example.Mapper.One", "example.Mapper.Two"} {
		if _, err := configuration.Statement(id); err != nil {
			t.Fatalf("Statement(%q) error = %v", id, err)
		}
	}
}

func TestXMLConfigurationRejectsDuplicateStatementAcrossMapperDocuments(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`
<configuration>
    <environments default="prod">
        <environment id="prod"><driver>mysql</driver><dataSource>dsn</dataSource></environment>
    </environments>
    <mappers>
        <mapper namespace="example.Mapper"><select id="Find">select 1</select></mapper>
        <mapper namespace="example.Mapper"><select id="Find">select 2</select></mapper>
    </mappers>
</configuration>`)},
	}

	_, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if err == nil || !strings.Contains(err.Error(), "duplicate statement id: example.Mapper.Find") {
		t.Fatalf("duplicate statement error = %v", err)
	}
}

func TestXMLConfigurationBuildsDynamicStatement(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {
			Data: []byte(`
<configuration>
    <environments default="prod">
        <environment id="prod">
            <driver>mysql</driver>
            <dataSource>dsn</dataSource>
        </environment>
    </environments>
    <mappers>
        <mapper namespace="example.UserMapper">
            <sql id="columns">id, name</sql>
            <select id="Find">
                <bind name="active" value="enabled"/>
                SELECT <include refid="columns"/> FROM users
                <where>
                    <if test="name != nil">name = #{name}</if>
                    <choose>
                        <when test="active">AND active = 1</when>
                        <otherwise>AND active = 0</otherwise>
                    </choose>
                    AND id IN
                    <foreach collection="ids" item="id" open="(" close=")" separator=",">
                        #{id}
                    </foreach>
                </where>
            </select>
        </mapper>
    </mappers>
</configuration>`),
		},
	}

	configuration, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if err != nil {
		t.Fatal(err)
	}
	statement, err := configuration.GetStatement("example.UserMapper.Find")
	if err != nil {
		t.Fatal(err)
	}

	query, args, err := statement.Build(
		driver.MySQLDriver{}.Translator(),
		eval.NewGenericParam(eval.H{
			"name":    "alice",
			"enabled": true,
			"ids":     []int{1, 2},
		}, ""),
	)
	if err != nil {
		t.Fatal(err)
	}
	query = strings.Join(strings.Fields(query), " ")

	for _, fragment := range []string{
		"SELECT id, name FROM users",
		"WHERE name = ?",
		"AND active = 1",
		"AND id IN (?,?)",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query %q does not contain %q", query, fragment)
		}
	}
	if len(args) != 3 || args[0] != "alice" || args[1] != 1 || args[2] != 2 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestXMLConfigurationResolvesCrossNamespaceSQL(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`
<configuration>
    <environments default="prod">
        <environment id="prod"><driver>mysql</driver><dataSource>dsn</dataSource></environment>
    </environments>
    <mappers>
        <mapper namespace="example.UserMapper">
            <select id="Find">SELECT <include refid="example.Shared.columns"/> FROM users</select>
        </mapper>
        <mapper namespace="example.Shared">
            <sql id="columns">id, name</sql>
        </mapper>
    </mappers>
</configuration>`)},
	}

	configuration, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if err != nil {
		t.Fatal(err)
	}
	statement, err := configuration.Statement("example.UserMapper.Find")
	if err != nil {
		t.Fatal(err)
	}
	query, _, err := statement.Build(driver.MySQLDriver{}.Translator(), eval.NewGenericParam(nil, ""))
	if err != nil {
		t.Fatal(err)
	}
	if normalized := strings.Join(strings.Fields(query), " "); normalized != "SELECT id, name FROM users" {
		t.Fatalf("unexpected query: %q", normalized)
	}
}

func TestXMLConfigurationRejectsInvalidExpression(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`
<configuration>
    <environments default="prod">
        <environment id="prod"><driver>mysql</driver><dataSource>dsn</dataSource></environment>
    </environments>
    <mappers>
        <mapper namespace="example.Mapper">
            <select id="One"><if test="id != ">select 1</if></select>
        </mapper>
    </mappers>
</configuration>`)},
	}
	_, err := NewXMLConfigurationWithFS(fsys, "juice.xml")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestXMLConfigurationIgnoreEnvironmentSkipsEnvironmentParsing(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`
<configuration>
    <environments default="prod">
        <environment id="prod" provider="missing">
            <maxOpenConnNum>not-a-number</maxOpenConnNum>
        </environment>
    </environments>
    <mappers>
        <mapper namespace="example.Mapper"><select id="One">select 1</select></mapper>
    </mappers>
</configuration>`)},
	}
	configured, err := compileXMLConfiguration(fsys, "juice.xml", true)
	if err != nil {
		t.Fatal(err)
	}
	if configured.DefaultSource() != "" {
		t.Fatalf("expected empty default source, got %q", configured.DefaultSource())
	}
	for name := range configured.Sources() {
		t.Fatalf("expected no sources, got %q", name)
	}
}
