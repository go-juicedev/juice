package juice

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

func TestConfigurationAdapterBuildsDynamicStatement(t *testing.T) {
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

func TestConfigurationAdapterRejectsInvalidExpression(t *testing.T) {
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
	configuration, err := newXMLConfigurationParser(fsys, "juice.xml", true)
	if err != nil {
		t.Fatal(err)
	}
	if configuration.(*xmlConfiguration).environments != nil {
		t.Fatalf("expected environments to be ignored, got %#v", configuration.Environments())
	}
}
