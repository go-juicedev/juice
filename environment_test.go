package juice

import (
	"fmt"
	"strings"
	"testing"
)

func TestEnvironmentsMethods_environment_test(t *testing.T) {
	env := &environments{envs: map[string]*Environment{"dev": {DataSource: "dsn"}}}
	if got, err := env.Use("dev"); err != nil || got == nil {
		t.Fatalf("expected env use success, got env=%v err=%v", got, err)
	}
	if _, err := env.Use("missing"); err == nil || !strings.Contains(err.Error(), "environment missing not found") {
		t.Fatalf("expected missing env error, got %v", err)
	}

	count := 0
	for range env.Iter() {
		count++
	}
	if count != 1 {
		t.Fatalf("expected one env in iterator, got %d", count)
	}

	if got, err := (OsEnvValueProvider{}).Get("prefix-${JUICE_TEST_ENV}-suffix"); err == nil || got != "prefix--suffix" {
		t.Fatalf("expected missing env substitution result with error, got %q err=%v", got, err)
	}

	if got, err := (EnvValueProviderFunc(func(key string) (string, error) {
		return fmt.Sprintf("value:%s", key), nil
	})).Get("abc"); err != nil || got != "value:abc" {
		t.Fatalf("unexpected EnvValueProviderFunc result got=%q err=%v", got, err)
	}

	customName := "custom_test_provider"
	RegisterEnvValueProvider(customName, EnvValueProviderFunc(func(key string) (string, error) { return "ok:" + key, nil }))
	if got, err := GetEnvValueProvider(customName).Get("x"); err != nil || got != "ok:x" {
		t.Fatalf("unexpected custom provider result got=%q err=%v", got, err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when provider name is empty")
		}
	}()
	RegisterEnvValueProvider("", EnvValueProviderFunc(func(key string) (string, error) { return key, nil }))
}

