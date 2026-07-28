package juice

import (
	"errors"
	"testing"
	"time"
)

func TestNewRuntimeConfigClonesInputs(t *testing.T) {
	sources := map[string]Source{
		"primary": {
			Driver:          "postgres",
			DSN:             "original",
			ConnMaxLifetime: time.Minute,
		},
	}
	settings := map[string]string{"debug": "false"}

	configuration, err := NewRuntimeConfig("primary", sources, settings)
	if err != nil {
		t.Fatalf("NewRuntimeConfig() error = %v", err)
	}

	sources["primary"] = Source{Driver: "mysql", DSN: "mutated"}
	settings["debug"] = "true"

	source, exists := configuration.Source("primary")
	if !exists {
		t.Fatal("primary source not found")
	}
	if source.Driver != "postgres" || source.DSN != "original" || source.ConnMaxLifetime != time.Minute {
		t.Fatalf("compiled source was mutated: %#v", source)
	}
	if configuration.Settings().Get("debug") != "false" {
		t.Fatalf("compiled settings were mutated: %q", configuration.Settings().Get("debug"))
	}

	source.DSN = "changed copy"
	stored, _ := configuration.Source("primary")
	if stored.DSN != "original" {
		t.Fatalf("Source returned mutable state: %#v", stored)
	}
}

func TestNewRuntimeConfigValidatesDefaultSource(t *testing.T) {
	tests := []struct {
		name          string
		defaultSource string
		sources       map[string]Source
		want          error
	}{
		{name: "no sources", defaultSource: "primary", want: errConfigurationEnvironmentsRequired},
		{name: "no default", sources: map[string]Source{"primary": {}}, want: errConfigurationDefaultEnvironmentMissing},
		{name: "unknown default", defaultSource: "missing", sources: map[string]Source{"primary": {}}, want: errConfigurationDefaultEnvironmentUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRuntimeConfig(tt.defaultSource, tt.sources, nil)
			if !errors.Is(err, tt.want) {
				t.Fatalf("NewRuntimeConfig() error = %v, want %v", err, tt.want)
			}
		})
	}
}
