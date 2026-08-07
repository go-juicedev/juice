package xml_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
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

func TestParserParseFileRejectsMissingInclude(t *testing.T) {
	fsys := fstest.MapFS{
		"juice.xml": {Data: []byte(`
<configuration>
    <mappers>
        <mapper namespace="example.Mapper">
            <select id="Find">SELECT <include refid="missing"/> FROM users</select>
        </mapper>
    </mappers>
</configuration>`)},
	}

	_, err := (&xmlparser.Parser{FS: fsys}).ParseFile("juice.xml")
	if err == nil || !strings.Contains(err.Error(), `SQL node "example.Mapper.missing" not found`) {
		t.Fatalf("unexpected error: %v", err)
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

func TestParseConfigurationRejectsMappersPrefix(t *testing.T) {
	_, err := xmlparser.New(strings.NewReader(`
<configuration>
    <mappers prefix="app"/>
</configuration>`))
	if err == nil || !strings.Contains(err.Error(), `attribute "prefix" is not allowed on <mappers>`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMapperDynamicNodes(t *testing.T) {
	mapperDocument, err := xmlparser.ParseMapper(strings.NewReader(`
<mapper namespace="example.UserMapper">
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

func TestParseMapperLinksForwardInclude(t *testing.T) {
	mapperDocument, err := xmlparser.ParseMapper(strings.NewReader(`
<mapper namespace="example.Mapper">
    <select id="Find">SELECT <include refid="projection"/> FROM users</select>
    <sql id="projection"><include refid="columns"/></sql>
    <sql id="columns">id, name</sql>
</mapper>`))
	if err != nil {
		t.Fatal(err)
	}

	query, _, err := mapperDocument.Statements[0].Node.Accept(
		driver.MySQLDriver{}.Translator(),
		eval.NewGenericParam(nil, ""),
	)
	if err != nil {
		t.Fatal(err)
	}
	if normalized := strings.Join(strings.Fields(query), " "); normalized != "SELECT id, name FROM users" {
		t.Fatalf("unexpected query: %q", normalized)
	}
}

func TestParseMapperRejectsMissingInclude(t *testing.T) {
	_, err := xmlparser.ParseMapper(strings.NewReader(`
<mapper namespace="example.Mapper">
    <select id="Find">SELECT <include refid="missing"/> FROM users</select>
</mapper>`))
	if err == nil || !strings.Contains(err.Error(), `SQL node "example.Mapper.missing" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMapperRejectsCyclicIncludes(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		cycle string
	}{
		{
			name:  "self reference",
			body:  `<sql id="a"><include refid="a"/></sql>`,
			cycle: "example.Mapper.a -> example.Mapper.a",
		},
		{
			name: "indirect cycle",
			body: `
<sql id="a"><include refid="b"/></sql>
<sql id="b"><include refid="c"/></sql>
<sql id="c"><include refid="a"/></sql>`,
			cycle: "example.Mapper.a -> example.Mapper.b -> example.Mapper.c -> example.Mapper.a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := xmlparser.ParseMapper(strings.NewReader(
				`<mapper namespace="example.Mapper">` + tt.body + `</mapper>`,
			))
			if err == nil || !strings.Contains(err.Error(), "cyclic SQL include: "+tt.cycle) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseConfigurationRejectsCrossNamespaceCyclicIncludes(t *testing.T) {
	_, err := xmlparser.New(strings.NewReader(`
<configuration>
    <mappers>
        <mapper namespace="example.First">
            <sql id="a"><include refid="example.Second.b"/></sql>
        </mapper>
        <mapper namespace="example.Second">
            <sql id="b"><include refid="example.First.a"/></sql>
        </mapper>
    </mappers>
</configuration>`))
	cycle := "example.First.a -> example.Second.b -> example.First.a"
	if err == nil || !strings.Contains(err.Error(), "cyclic SQL include: "+cycle) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseMapperRejectsDocumentAttributes(t *testing.T) {
	_, err := xmlparser.ParseMapper(strings.NewReader(`
<mapper namespace="example.UserMapper" unknown="value">
    <select id="Find">select 1</select>
</mapper>`))
	if err == nil || !strings.Contains(err.Error(), `attribute "unknown" is not allowed on <mapper>`) {
		t.Fatalf("unexpected error: %v", err)
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
