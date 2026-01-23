package juice

import (
	"encoding/xml"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestXMLParseError(t *testing.T) {
	tests := []struct {
		name      string
		xmlErr    *XMLParseError
		wantInErr []string
	}{
		{
			name: "error with all fields",
			xmlErr: &XMLParseError{
				Namespace:  "com.example.UserMapper",
				XMLContent: `<if test="id != null">`,
				Err:        errors.New("test attribute is invalid"),
			},
			wantInErr: []string{
				"XML parse error",
				"namespace 'com.example.UserMapper'",
				`<if test="id != null">`,
				"test attribute is invalid",
			},
		},
		{
			name: "error with namespace only",
			xmlErr: &XMLParseError{
				Namespace: "com.example.OrderMapper",
				Err:       errors.New("missing required attribute"),
			},
			wantInErr: []string{
				"XML parse error",
				"namespace 'com.example.OrderMapper'",
				"missing required attribute",
			},
		},
		{
			name: "error with xml content only",
			xmlErr: &XMLParseError{
				XMLContent: `<foreach collection="items" item="item">`,
				Err:        errors.New("invalid collection"),
			},
			wantInErr: []string{
				"XML parse error",
				`<foreach collection="items" item="item">`,
				"invalid collection",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMsg := tt.xmlErr.Error()
			for _, want := range tt.wantInErr {
				if !strings.Contains(errMsg, want) {
					t.Errorf("XMLParseError.Error() = %v, want to contain %v", errMsg, want)
				}
			}
		})
	}
}

func TestXMLParseErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	xmlErr := &XMLParseError{
		Namespace: "test.namespace",
		Err:       originalErr,
	}

	if !errors.Is(xmlErr, originalErr) {
		t.Errorf("XMLParseError should unwrap to original error")
	}
}

func TestBuildXMLContent(t *testing.T) {
	tests := []struct {
		name  string
		token StartElement
		want  string
	}{
		{
			name: "simple element",
			token: StartElement{
				Name: Name{Local: "if"},
				Attr: []Attr{
					{Name: Name{Local: "test"}, Value: "id != null"},
				},
			},
			want: `<if test="id != null">`,
		},
		{
			name: "element with multiple attributes",
			token: StartElement{
				Name: Name{Local: "foreach"},
				Attr: []Attr{
					{Name: Name{Local: "collection"}, Value: "items"},
					{Name: Name{Local: "item"}, Value: "item"},
					{Name: Name{Local: "separator"}, Value: ","},
				},
			},
			want: `<foreach collection="items" item="item" separator=",">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert test struct to xml.StartElement
			var attrs []xml.Attr
			for _, a := range tt.token.Attr {
				attrs = append(attrs, xml.Attr{
					Name:  xml.Name{Local: a.Name.Local},
					Value: a.Value,
				})
			}
			xmlToken := xml.StartElement{
				Name: xml.Name{Local: tt.token.Name.Local},
				Attr: attrs,
			}

			got := buildXMLContent(xmlToken)
			if got != tt.want {
				t.Errorf("buildXMLContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper types for testing
type Name struct {
	Local string
}

type Attr struct {
	Name  Name
	Value string
}

type StartElement struct {
	Name Name
	Attr []Attr
}

// TestXMLParseErrorIntegration tests the XML parsing error reporting with actual XML files
func TestXMLParseErrorIntegration(t *testing.T) {
	fsys := os.DirFS(".")

	tests := []struct {
		name          string
		xmlContent    string
		wantErrType   *XMLParseError
		wantContains  []string
		wantNamespace string
	}{
		{
			name: "missing select id attribute",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="test.MissingIdMapper">
    <select>
        SELECT * FROM users
    </select>
</mapper>`,
			wantErrType:   &XMLParseError{},
			wantNamespace: "test.MissingIdMapper",
			wantContains: []string{
				"XML parse error",
				"namespace 'test.MissingIdMapper'",
				"<select>",
				"id is required",
			},
		},
		{
			name: "missing if test attribute",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="test.MissingTestMapper">
    <select id="getUser">
        SELECT * FROM users
        <if>
            WHERE id = #{id}
        </if>
    </select>
</mapper>`,
			wantErrType:   &XMLParseError{},
			wantNamespace: "test.MissingTestMapper",
			wantContains: []string{
				"XML parse error",
				"namespace 'test.MissingTestMapper'",
				"<if>",
				"test",
			},
		},
		{
			name: "missing foreach item attribute",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="test.MissingItemMapper">
    <select id="getUsersByIds">
        SELECT * FROM users WHERE id IN
        <foreach collection="ids" open="(" close=")" separator=",">
            #{id}
        </foreach>
    </select>
</mapper>`,
			wantErrType:   &XMLParseError{},
			wantNamespace: "test.MissingItemMapper",
			wantContains: []string{
				"XML parse error",
				"namespace 'test.MissingItemMapper'",
				`<foreach collection="ids" open="(" close=")" separator=",">`,
				"item",
			},
		},
		{
			name: "invalid expression in if test",
			xmlContent: `<?xml version="1.0" encoding="UTF-8"?>
<mapper namespace="test.InvalidExpressionMapper">
    <select id="getUser">
        SELECT * FROM users
        <if test="id != ">
            WHERE id = #{id}
        </if>
    </select>
</mapper>`,
			wantErrType:   &XMLParseError{},
			wantNamespace: "test.InvalidExpressionMapper",
			wantContains: []string{
				"XML parse error",
				"namespace 'test.InvalidExpressionMapper'",
				`<if test="id != ">`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary XML file
			tmpFile, err := os.CreateTemp("", "test_mapper_*.xml")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			if _, err := tmpFile.WriteString(tt.xmlContent); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Parse the mapper
			parser := &XMLMappersElementParser{parser: &XMLParser{FS: fsys}}
			file, err := os.Open(tmpFile.Name())
			if err != nil {
				t.Fatalf("Failed to open temp file: %v", err)
			}
			defer file.Close()

			_, parseErr := parser.parseMapperByReader(file)

			// Check if we got an XMLParseError
			if parseErr == nil {
				t.Fatalf("Expected an error but got nil")
			}

			var xmlParseErr *XMLParseError
			if !errors.As(parseErr, &xmlParseErr) {
				t.Fatalf("Expected XMLParseError but got: %T: %v", parseErr, parseErr)
			}

			// Check namespace
			if xmlParseErr.Namespace != tt.wantNamespace {
				t.Errorf("Expected namespace %q but got %q", tt.wantNamespace, xmlParseErr.Namespace)
			}

			// Check error message contains expected strings
			errMsg := parseErr.Error()
			for _, want := range tt.wantContains {
				if !strings.Contains(errMsg, want) {
					t.Errorf("Error message does not contain %q\nGot: %s", want, errMsg)
				}
			}

			t.Logf("Error message: %s", errMsg)
		})
	}
}
