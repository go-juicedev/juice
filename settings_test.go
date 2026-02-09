package juice

import (
	"errors"
	"testing"
)

type textUnmarshalerStub struct {
	value string
	err   error
}

func (t *textUnmarshalerStub) UnmarshalText(text []byte) error {
	t.value = string(text)
	return t.err
}

func TestStringValue_Conversions_settings_test(t *testing.T) {
	if !StringValue("true").Bool() {
		t.Fatalf("expected true")
	}

	if StringValue("not-bool").Bool() {
		t.Fatalf("expected invalid bool to be false")
	}

	if got := StringValue("123").Int64(); got != 123 {
		t.Fatalf("expected int64 123, got %d", got)
	}

	if got := StringValue("bad-int").Int64(); got != 0 {
		t.Fatalf("expected invalid int64 to be 0, got %d", got)
	}

	if got := StringValue("456").Uint64(); got != 456 {
		t.Fatalf("expected uint64 456, got %d", got)
	}

	if got := StringValue("bad-uint").Uint64(); got != 0 {
		t.Fatalf("expected invalid uint64 to be 0, got %d", got)
	}

	if got := StringValue("3.14").Float64(); got != 3.14 {
		t.Fatalf("expected float64 3.14, got %v", got)
	}

	if got := StringValue("bad-float").Float64(); got != 0 {
		t.Fatalf("expected invalid float64 to be 0, got %v", got)
	}

	if got := StringValue("hello").String(); got != "hello" {
		t.Fatalf("expected string hello, got %q", got)
	}
}

func TestStringValue_Unmarshaler_settings_test(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var target textUnmarshalerStub
		if err := StringValue("value").Unmarshaler(&target); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if target.value != "value" {
			t.Fatalf("expected unmarshaled value, got %q", target.value)
		}
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("unmarshal failed")
		target := textUnmarshalerStub{err: want}
		err := StringValue("value").Unmarshaler(&target)
		if !errors.Is(err, want) {
			t.Fatalf("expected %v, got %v", want, err)
		}
	})
}

func TestKeyValueSettingProvider_Get_settings_test(t *testing.T) {
	provider := keyValueSettingProvider{
		"name": "juice",
	}

	if got := provider.Get("name"); got != "juice" {
		t.Fatalf("expected juice, got %q", got)
	}

	if got := provider.Get("missing"); got != "" {
		t.Fatalf("expected empty value for missing key, got %q", got)
	}
}
