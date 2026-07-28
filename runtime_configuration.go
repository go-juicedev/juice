/*
Copyright 2026 eatmoreapple

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package juice

import (
	"fmt"
	"iter"
	"maps"
)

// SourceProvider provides immutable database source definitions.
type SourceProvider interface {
	// DefaultSource returns the source selected when an Engine starts.
	DefaultSource() string

	// Source returns a source definition by name.
	Source(name string) (Source, bool)

	// Sources iterates over all source definitions.
	Sources() iter.Seq2[string, Source]
}

// RuntimeConfig contains the settings and database source definitions needed
// to start an Engine. It owns no database connections.
type RuntimeConfig struct {
	defaultSource string
	sources       map[string]Source
	settings      keyValueSettingProvider
}

// NewRuntimeConfig creates an immutable runtime configuration.
func NewRuntimeConfig(defaultSource string, sources map[string]Source, settings map[string]string) (*RuntimeConfig, error) {
	if len(sources) == 0 {
		return nil, errConfigurationEnvironmentsRequired
	}
	if defaultSource == "" {
		return nil, errConfigurationDefaultEnvironmentMissing
	}
	if _, exists := sources[defaultSource]; !exists {
		return nil, fmt.Errorf("%w: %s", errConfigurationDefaultEnvironmentUnknown, defaultSource)
	}

	compiledSettings := make(keyValueSettingProvider, len(settings))
	for name, value := range settings {
		compiledSettings[name] = StringValue(value)
	}
	return &RuntimeConfig{
		defaultSource: defaultSource,
		sources:       maps.Clone(sources),
		settings:      compiledSettings,
	}, nil
}

// DefaultSource returns the source selected when an Engine starts.
func (c *RuntimeConfig) DefaultSource() string {
	if c == nil {
		return ""
	}
	return c.defaultSource
}

// Source returns a source definition by name.
func (c *RuntimeConfig) Source(name string) (Source, bool) {
	if c == nil {
		return Source{}, false
	}
	source, exists := c.sources[name]
	return source, exists
}

// Sources iterates over source definitions. Values are returned by copy.
func (c *RuntimeConfig) Sources() iter.Seq2[string, Source] {
	if c == nil {
		return maps.All(map[string]Source(nil))
	}
	return maps.All(c.sources)
}

// Settings returns the runtime settings.
func (c *RuntimeConfig) Settings() SettingProvider {
	if c == nil {
		return keyValueSettingProvider(nil)
	}
	return c.settings
}

var _ RuntimeConfiguration = (*RuntimeConfig)(nil)
