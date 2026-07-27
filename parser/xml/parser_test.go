package xml_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-juicedev/juice/parser"
	xmlparser "github.com/go-juicedev/juice/parser/xml"
)

func TestParserParseFileLoadsMapperSources(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`
<configuration>
    <mappers pattern="mappers/*.xml">
        <mapper resource="single.xml"/>
		<mapper namespace="inline"><select id="One">select 1</select></mapper>
    </mappers>
</configuration>`)},
		"single.xml":         {Data: []byte(`<mapper namespace="single"><select id="One">select 1</select></mapper>`)},
		"mappers/first.xml":  {Data: []byte(`<mapper namespace="first"><select id="One">select 1</select></mapper>`)},
		"mappers/second.xml": {Data: []byte(`<mapper namespace="second"><select id="One">select 1</select></mapper>`)},
	}

	document, err := (&xmlparser.Parser{FS: fsys}).ParseFile("juice.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Mappers) != 4 {
		t.Fatalf("unexpected mappers: %#v", document.Mappers)
	}
	if document.Mappers[0].Namespace != "first" || document.Mappers[2].Namespace != "single" || document.Mappers[3].Namespace != "inline" {
		t.Fatalf("unexpected mapper order: %#v", document.Mappers)
	}
}

func TestParseConfigurationDocument(t *testing.T) {
	document, err := xmlparser.New(strings.NewReader(`
<configuration>
    <settings>
        <setting name="debug" value="true"/>
    </settings>
    <environments default="prod">
        <environment id="prod">
            <driver>sqlite3</driver>
            <dataSource>app.db</dataSource>
            <maxOpenConnNum>20</maxOpenConnNum>
        </environment>
    </environments>
    <mappers pattern="mappers/*.xml">
        <mapper resource="mappers/user.xml"/>
        <mapper url="https://example.com/order.xml"/>
        <mapper namespace="example.Inline">
            <select id="Ping">select 1</select>
        </mapper>
    </mappers>
</configuration>`))
	if err != nil {
		t.Fatal(err)
	}

	if document.Settings["debug"] != "true" {
		t.Fatalf("unexpected settings: %#v", document.Settings)
	}
	if document.Environments.Default != "prod" || len(document.Environments.Items) != 1 {
		t.Fatalf("unexpected environments: %#v", document.Environments)
	}
	environment := document.Environments.Items[0]
	if environment.Driver != "sqlite3" || environment.DataSource != "app.db" || environment.MaxOpenConns != "20" {
		t.Fatalf("unexpected environment: %#v", environment)
	}
	if len(document.Mappers) != 1 || document.Mappers[0].Namespace != "example.Inline" {
		t.Fatalf("unexpected inline mappers: %#v", document.Mappers)
	}
}

func TestParseMapperDynamicNodes(t *testing.T) {
	mapperDocument, err := xmlparser.ParseMapper(strings.NewReader(`
<mapper namespace="example.UserMapper" timeout="1000">
    <sql id="columns">id, name</sql>
    <select id="Find" dataSource="replica">
        select <include refid="columns"><property name="prefix" value="u"/></include>
        from users
        <where>
            <if test="name != nil">name = #{name}</if>
            <foreach collection="ids" item="id" open="(" close=")" separator=",">
                #{id}
            </foreach>
            <choose>
                <when test="active">active = 1</when>
                <otherwise>active = 0</otherwise>
            </choose>
        </where>
    </select>
</mapper>`))
	if err != nil {
		t.Fatal(err)
	}

	if mapperDocument.Namespace != "example.UserMapper" {
		t.Fatalf("unexpected namespace: %s", mapperDocument.Namespace)
	}
	if len(mapperDocument.Statements) != 1 {
		t.Fatalf("unexpected statements: %#v", mapperDocument.Statements)
	}
	statement := mapperDocument.Statements[0]
	if statement.Action != parser.Select || statement.ID != "Find" || statement.Attributes["dataSource"] != "replica" {
		t.Fatalf("unexpected statement: %#v", statement)
	}

	if statement.Node == nil {
		t.Fatal("expected the XML backend to return an executable node")
	}
}

func TestParseMapperRejectsMissingStatementID(t *testing.T) {
	_, err := xmlparser.ParseMapper(strings.NewReader(`
<mapper namespace="example.UserMapper">
    <select>select 1</select>
</mapper>`))
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "attribute \"id\" is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParserParseFileLoadsRemoteMapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`<mapper namespace="remote"><select id="One">select 1</select></mapper>`))
	}))
	defer server.Close()

	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`<configuration><mappers><mapper url="` + server.URL + `"/></mappers></configuration>`)},
	}
	document, err := (&xmlparser.Parser{FS: fsys}).ParseFile("juice.xml")
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Mappers) != 1 || document.Mappers[0].Namespace != "remote" {
		t.Fatalf("unexpected remote mapper: %#v", document.Mappers)
	}
}

func TestParserParseFileRejectsRemoteHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`<configuration><mappers><mapper url="` + server.URL + `"/></mappers></configuration>`)},
	}
	_, err := (&xmlparser.Parser{FS: fsys}).ParseFile("juice.xml")
	if !errors.Is(err, xmlparser.ErrUnexpectedHTTPStatus) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMapperRejectsUnclosedElements(t *testing.T) {
	tests := []string{
		`<mapper namespace="example.Mapper"><select id="One">select 1`,
		`<mapper namespace="example.Mapper"><select id="One">select 1</select>`,
	}
	for _, input := range tests {
		_, err := xmlparser.ParseMapper(strings.NewReader(input))
		if err == nil || !strings.Contains(err.Error(), "not closed") {
			t.Fatalf("unexpected error for %q: %v", input, err)
		}
		var parseError *xmlparser.ParseError
		if !errors.As(err, &parseError) {
			t.Fatalf("expected ParseError, got %T", err)
		}
	}
}

func TestParseMapperPreservesTrimText(t *testing.T) {
	mapperDocument, err := xmlparser.ParseMapper(strings.NewReader(`
<mapper namespace="example.Mapper">
    <select id="One">
        select * from users
        <trim prefix=" WHERE " prefixOverrides="AND ">
            AND id = #{id}
        </trim>
    </select>
</mapper>`))
	if err != nil {
		t.Fatal(err)
	}
	if mapperDocument.Statements[0].Node == nil {
		t.Fatal("expected the XML backend to return an executable trim node")
	}
}
