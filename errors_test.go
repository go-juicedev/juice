package juice

import (
	"errors"
	"strings"
	"testing"
)

func TestErrorTypes_errors_test(t *testing.T) {
	if got := (&nodeUnclosedError{nodeName: "if"}).Error(); got != "node if is not closed" {
		t.Fatalf("unexpected nodeUnclosedError message: %q", got)
	}
	if got := (&nodeAttributeConflictError{nodeName: "set", attrName: "x"}).Error(); got != "node set has conflicting attribute x" {
		t.Fatalf("unexpected nodeAttributeConflictError message: %q", got)
	}

	xmlErr := &XMLParseError{Namespace: "ns", XMLContent: "<a>", Err: errors.New("bad")}
	if got := xmlErr.Error(); !strings.Contains(got, "XML parse error") || !strings.Contains(got, "namespace 'ns'") || !strings.Contains(got, "<a>") || !strings.Contains(got, "bad") {
		t.Fatalf("unexpected XMLParseError message: %q", got)
	}

	if !errors.Is(xmlErr, xmlErr.Err) {
		t.Fatalf("expected XMLParseError unwrap to underlying error")
	}

	xmlErr = &XMLParseError{Err: nil}
	if got := xmlErr.Error(); got != "XML parse error" {
		t.Fatalf("unexpected minimal XMLParseError message: %q", got)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected unreachable to panic")
		}
	}()
	_ = unreachable()
}

