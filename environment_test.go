package juice

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
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

	t.Setenv("JUICE_TEST_ENV", "value")
	t.Setenv("JUICE.TEST.ENV", "dot")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "missing braced variable",
			input: "prefix-${JUICE_TEST_ENV_MISSING}-suffix",
			want:  "prefix--suffix",
		},
		{
			name:  "braced variable",
			input: "prefix-${JUICE_TEST_ENV}-suffix",
			want:  "prefix-value-suffix",
		},
		{
			name:  "dotted variable",
			input: "prefix-${JUICE.TEST.ENV}-suffix",
			want:  "prefix-dot-suffix",
		},
		{
			name:  "shell style variable",
			input: "prefix-$JUICE_TEST_ENV-suffix",
			want:  "prefix-value-suffix",
		},
		{
			name:  "spaced braced variable",
			input: "prefix-${ JUICE_TEST_ENV }-suffix",
			want:  "prefix--suffix",
		},
		{
			name:  "invalid braced variable",
			input: "prefix-${}-suffix",
			want:  "prefix--suffix",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (OsEnvValueProvider{}).Get(tt.input)
			if err != nil {
				t.Fatalf("OsEnvValueProvider.Get() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("OsEnvValueProvider.Get() = %q, want %q", got, tt.want)
			}
		})
	}

	if got, err := (EnvValueProviderFunc(func(key string) (string, error) {
		return fmt.Sprintf("value:%s", key), nil
	})).Get("abc"); err != nil || got != "value:abc" {
		t.Fatalf("unexpected EnvValueProviderFunc result got=%q err=%v", got, err)
	}

	customName := "custom_test_provider"
	if err := RegisterEnvValueProvider(customName, EnvValueProviderFunc(func(key string) (string, error) { return "ok:" + key, nil })); err != nil {
		t.Fatalf("RegisterEnvValueProvider() error = %v", err)
	}
	if got, err := LookupEnvValueProvider(customName).Get("x"); err != nil || got != "ok:x" {
		t.Fatalf("unexpected custom provider result got=%q err=%v", got, err)
	}

	if err := RegisterEnvValueProvider("", EnvValueProviderFunc(func(key string) (string, error) { return key, nil })); err == nil {
		t.Fatalf("expected error when provider name is empty")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when provider name is empty")
		}
	}()
	MustRegisterEnvValueProvider("", EnvValueProviderFunc(func(key string) (string, error) { return key, nil }))
}

func TestEnvValueProviderRegistryConcurrentAccess_environment_test(t *testing.T) {
	const workers = 8
	const iterations = 1000

	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < workers; i++ {
		name := "concurrent_test_provider_" + strconv.Itoa(i)
		provider := EnvValueProviderFunc(func(key string) (string, error) { return name + ":" + key, nil })

		wg.Add(2)

		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				if err := RegisterEnvValueProvider(name, provider); err != nil {
					t.Errorf("RegisterEnvValueProvider(%q) error = %v", name, err)
					return
				}
			}
		}()

		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				got, err := LookupEnvValueProvider(name).Get("x")
				if err != nil {
					t.Errorf("LookupEnvValueProvider(%q).Get() error = %v", name, err)
					return
				}
				if got != name+":x" && got != "x" {
					t.Errorf("LookupEnvValueProvider(%q).Get() = %q", name, got)
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
