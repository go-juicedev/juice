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
	gotoken "go/token"
	"maps"
	"slices"
	"strconv"
	"time"

	configparser "github.com/go-juicedev/juice/parser"
)

func adaptSettings(source map[string]string) keyValueSettingProvider {
	settings := make(keyValueSettingProvider, len(source))
	for name, value := range source {
		settings[name] = StringValue(value)
	}
	return settings
}

func resolveEnvironmentString(provider EnvValueProvider, value string) (string, error) {
	if value == "" {
		return "", nil
	}
	return provider.Get(value)
}

func resolveEnvironmentInt(provider EnvValueProvider, value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	resolved, err := provider.Get(value)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(resolved)
}

func adaptRuntimeConfig(document *configparser.Document, lookup EnvValueProviderLookup) (*RuntimeConfig, error) {
	compiled := &RuntimeConfig{
		defaultSource: document.Environments.Default,
		sources:       make(map[string]Source, len(document.Environments.Items)),
		settings:      adaptSettings(document.Settings),
	}
	for _, item := range document.Environments.Items {
		if item.ID == "" {
			return nil, fmt.Errorf("environment id is required")
		}
		if !gotoken.IsIdentifier(item.ID) {
			return nil, fmt.Errorf("environment id is invalid: %s", item.ID)
		}
		if _, exists := compiled.sources[item.ID]; exists {
			return nil, fmt.Errorf("duplicate environment id: %s", item.ID)
		}

		providerName := item.Attributes["provider"]
		provider, ok := lookup(providerName)
		if !ok || provider == nil {
			return nil, fmt.Errorf("%w: %s", errEnvValueProviderNotFound, providerName)
		}

		var err error
		var source Source
		if source.Driver, err = resolveEnvironmentString(provider, item.Driver); err != nil {
			return nil, err
		}
		if source.DSN, err = resolveEnvironmentString(provider, item.DataSource); err != nil {
			return nil, err
		}
		if source.MaxIdleConns, err = resolveEnvironmentInt(provider, item.MaxIdleConns); err != nil {
			return nil, err
		}
		if source.MaxOpenConns, err = resolveEnvironmentInt(provider, item.MaxOpenConns); err != nil {
			return nil, err
		}
		maxConnLifetime, err := resolveEnvironmentInt(provider, item.ConnMaxLifetime)
		if err != nil {
			return nil, err
		}
		maxIdleConnLifetime, err := resolveEnvironmentInt(provider, item.ConnMaxIdleLifetime)
		if err != nil {
			return nil, err
		}
		source.ConnMaxLifetime = time.Duration(maxConnLifetime) * time.Second
		source.ConnMaxIdleTime = time.Duration(maxIdleConnLifetime) * time.Second
		compiled.sources[item.ID] = source
	}
	return compiled, nil
}

func compileMappedStatement(namespace string, source configparser.Statement) (*mappedStatement, error) {
	if source.ID == "" {
		return nil, fmt.Errorf("statement id is required in mapper %s", namespace)
	}

	var actions = []configparser.Action{
		configparser.Delete,
		configparser.Insert,
		configparser.Update,
		configparser.Select,
	}

	if !slices.Contains(actions, source.Action) {
		return nil, fmt.Errorf("invalid action %q for statement %s.%s", source.Action, namespace, source.ID)
	}

	return &mappedStatement{
		action: source.Action,
		script: source.Node,
		attrs:  maps.Clone(source.Attributes),
		id:     newStatementID(namespace, source.ID),
	}, nil
}

func compileStatementCatalog(mappers []configparser.Mapper) (*statementCatalog, error) {
	compiled := newStatementCatalog()

	for _, mapperDocument := range mappers {
		if mapperDocument.Namespace == "" {
			return nil, fmt.Errorf("mapper namespace is required")
		}
		namespace := mapperDocument.Namespace

		for _, statementDocument := range mapperDocument.Statements {
			statement, err := compileMappedStatement(namespace, statementDocument)
			if err != nil {
				return nil, err
			}
			if err := compiled.add(statement); err != nil {
				return nil, err
			}
		}
	}
	return compiled, nil
}

// CompileOptions controls how a parsed document is compiled.
type CompileOptions struct {
	// Backend provides syntax-specific script compilation for the resulting Engine.
	Backend configparser.Backend

	// IgnoreEnvironment skips environment compilation and validation.
	IgnoreEnvironment bool

	// EnvValueProviderLookup resolves providers referenced by environments.
	// LookupEnvValueProvider is used when this field is nil.
	EnvValueProviderLookup EnvValueProviderLookup
}

// Compile validates and compiles a parsed document into an immutable artifact.
// The caller must not mutate Nodes retained by the document after calling Compile.
// Compile does not open database connections.
func Compile(document *configparser.Document, options CompileOptions) (*CompiledConfig, error) {
	if document == nil {
		return nil, errConfigurationRequired
	}
	lookup := options.EnvValueProviderLookup
	if lookup == nil {
		lookup = LookupEnvValueProvider
	}

	compiled := &CompiledConfig{backend: options.Backend}

	var runtime *RuntimeConfig
	if !options.IgnoreEnvironment {
		var err error
		runtime, err = adaptRuntimeConfig(document, lookup)
		if err != nil {
			return nil, err
		}
	} else {
		runtime = &RuntimeConfig{
			sources:  make(map[string]Source),
			settings: adaptSettings(document.Settings),
		}
	}
	compiled.runtime = runtime

	catalog, err := compileStatementCatalog(document.Mappers)
	if err != nil {
		return nil, err
	}
	compiled.catalog = catalog

	if err := compiled.validate(options.IgnoreEnvironment); err != nil {
		return nil, err
	}
	return compiled, nil
}
