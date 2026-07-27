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
	"strconv"

	"github.com/go-juicedev/juice/node"
	configparser "github.com/go-juicedev/juice/parser"
	juicesql "github.com/go-juicedev/juice/sql"
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

func adaptEnvironments(source configparser.Environments) (*environments, error) {
	if !source.Present {
		return nil, nil
	}

	compiled := &environments{
		attr: map[string]string{"default": source.Default},
		envs: make(map[string]*Environment, len(source.Items)),
	}
	for _, item := range source.Items {
		if item.ID == "" {
			return nil, fmt.Errorf("environment id is required")
		}
		if !gotoken.IsIdentifier(item.ID) {
			return nil, fmt.Errorf("environment id is invalid: %s", item.ID)
		}
		if _, exists := compiled.envs[item.ID]; exists {
			return nil, fmt.Errorf("duplicate environment id: %s", item.ID)
		}

		environment := &Environment{attrs: maps.Clone(item.Attributes)}
		environment.setAttr("id", item.ID)
		provider, err := environment.provider()
		if err != nil {
			return nil, err
		}

		if environment.Driver, err = resolveEnvironmentString(provider, item.Driver); err != nil {
			return nil, err
		}
		if environment.DataSource, err = resolveEnvironmentString(provider, item.DataSource); err != nil {
			return nil, err
		}
		if environment.MaxIdleConnNum, err = resolveEnvironmentInt(provider, item.MaxIdleConns); err != nil {
			return nil, err
		}
		if environment.MaxOpenConnNum, err = resolveEnvironmentInt(provider, item.MaxOpenConns); err != nil {
			return nil, err
		}
		if environment.MaxConnLifetime, err = resolveEnvironmentInt(provider, item.ConnMaxLifetime); err != nil {
			return nil, err
		}
		if environment.MaxIdleConnLifetime, err = resolveEnvironmentInt(provider, item.ConnMaxIdleLifetime); err != nil {
			return nil, err
		}
		compiled.envs[item.ID] = environment
	}
	return compiled, nil
}

func adaptMapper(mapper *Mapper, source configparser.Mapper) error {
	for _, statementDocument := range source.Statements {
		if _, exists := mapper.statements[statementDocument.ID]; exists {
			return fmt.Errorf("duplicate statement id: %s", statementDocument.ID)
		}
		statement := &mappedStatement{
			mapper: mapper,
			action: juicesql.Action(statementDocument.Action),
			Nodes:  node.Group{statementDocument.Node},
			attrs:  maps.Clone(statementDocument.Attributes),
			id:     statementDocument.ID,
		}
		statement.name = statement.lazyName()
		mapper.statements[statement.id] = statement
	}
	return nil
}

func adaptMappers(configuration Configuration, document *configparser.Document) (*Mappers, error) {
	compiled := &Mappers{
		attrs: maps.Clone(document.MapperAttributes),
		cfg:   configuration,
	}
	for _, mapperDocument := range document.Mappers {
		mapper := &Mapper{
			namespace:  mapperDocument.Namespace,
			attrs:      maps.Clone(mapperDocument.Attributes),
			statements: make(map[string]*mappedStatement, len(mapperDocument.Statements)),
		}
		if err := compiled.setMapper(mapper.namespace, mapper); err != nil {
			return nil, err
		}
		if err := adaptMapper(mapper, mapperDocument); err != nil {
			return nil, err
		}
	}
	return compiled, nil
}

func adaptConfigurationDocument(document *configparser.Document, ignoreEnv bool) (Configuration, error) {
	if document == nil {
		return nil, errConfigurationRequired
	}

	configuration := &xmlConfiguration{
		settings: adaptSettings(document.Settings),
	}

	environments, err := adaptEnvironments(document.Environments)
	if err != nil {
		return nil, err
	}
	configuration.environments = environments

	mappers, err := adaptMappers(configuration, document)
	if err != nil {
		return nil, err
	}
	configuration.mappers = mappers

	if err := configuration.validate(ignoreEnv); err != nil {
		return nil, err
	}
	return configuration, nil
}
