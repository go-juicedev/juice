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
	"strings"

	"github.com/go-juicedev/juice/eval"
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

func adaptBindNode(source configparser.BindNode) (*node.BindNode, error) {
	compiled := &node.BindNode{Name: source.Name}
	if err := compiled.Parse(source.Value); err != nil {
		return nil, err
	}
	return compiled, nil
}

func splitOverrides(value string) []string {
	if value == "" {
		return nil
	}
	values := strings.Split(value, "|")
	for index := range values {
		values[index] = strings.TrimSpace(values[index])
	}
	return values
}
func adaptNodeGroup(source []configparser.Node, mapper *Mapper) (node.Group, node.BindNodeGroup, error) {
	nodes := make(node.Group, 0, len(source))
	var bindings node.BindNodeGroup
	for _, sourceNode := range source {
		if binding, ok := sourceNode.(configparser.BindNode); ok {
			compiled, err := adaptBindNode(binding)
			if err != nil {
				return nil, nil, err
			}
			bindings = append(bindings, compiled)
			continue
		}
		compiled, err := adaptNode(sourceNode, mapper)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, compiled)
	}
	return nodes, bindings, nil
}

func adaptTextNode(source configparser.TextNode) (node.Node, error) {
	return node.NewTextNode(source.Text), nil
}

func adaptIfNode(source configparser.IfNode, mapper *Mapper) (node.Node, error) {
	nodes, bindings, err := adaptNodeGroup(source.Children, mapper)
	if err != nil {
		return nil, err
	}
	compiled := &node.ConditionNode{Nodes: nodes, BindNodes: bindings}
	if err := compiled.Parse(source.Test); err != nil {
		return nil, err
	}
	return compiled, nil
}

func adaptForeachNode(source configparser.ForeachNode, mapper *Mapper) (node.Node, error) {
	nodes, bindings, err := adaptNodeGroup(source.Children, mapper)
	if err != nil {
		return nil, err
	}
	collection := source.Collection
	if collection == "" {
		collection = eval.DefaultParamKey()
	}
	return &node.ForeachNode{
		Collection: collection,
		Nodes:      nodes,
		Item:       source.Item,
		Index:      source.Index,
		Open:       source.Open,
		Close:      source.Close,
		Separator:  source.Separator,
		BindNodes:  bindings,
	}, nil
}

func adaptTrimNode(source configparser.TrimNode, mapper *Mapper) (node.Node, error) {
	nodes, bindings, err := adaptNodeGroup(source.Children, mapper)
	if err != nil {
		return nil, err
	}
	return &node.TrimNode{
		Nodes:           nodes,
		Prefix:          source.Prefix,
		Suffix:          source.Suffix,
		PrefixOverrides: splitOverrides(source.PrefixOverrides),
		SuffixOverrides: splitOverrides(source.SuffixOverrides),
		BindNodes:       bindings,
	}, nil
}

func adaptWhereNode(source configparser.WhereNode, mapper *Mapper) (node.Node, error) {
	nodes, bindings, err := adaptNodeGroup(source.Children, mapper)
	if err != nil {
		return nil, err
	}
	return &node.WhereNode{Nodes: nodes, BindNodes: bindings}, nil
}

func adaptSetNode(source configparser.SetNode, mapper *Mapper) (node.Node, error) {
	nodes, bindings, err := adaptNodeGroup(source.Children, mapper)
	if err != nil {
		return nil, err
	}
	return &node.SetNode{Nodes: nodes, BindNodes: bindings}, nil
}

func adaptIncludeNode(source configparser.IncludeNode, mapper *Mapper) (node.Node, error) {
	include := node.NewIncludeNode(nil, mapper, source.RefID)
	if len(source.Properties) == 0 {
		return include, nil
	}
	properties := make(eval.H, len(source.Properties))
	for name, value := range source.Properties {
		properties[name] = value
	}
	return include.WithProperties(properties), nil
}

func adaptChooseNode(source configparser.ChooseNode, mapper *Mapper) (node.Node, error) {
	compiled := &node.ChooseNode{}
	for _, binding := range source.Bindings {
		bindNode, err := adaptBindNode(binding)
		if err != nil {
			return nil, err
		}
		compiled.BindNodes = append(compiled.BindNodes, bindNode)
	}
	for _, when := range source.Whens {
		nodes, bindings, err := adaptNodeGroup(when.Children, mapper)
		if err != nil {
			return nil, err
		}
		whenNode := &node.ConditionNode{Nodes: nodes, BindNodes: bindings}
		if err := whenNode.Parse(when.Test); err != nil {
			return nil, err
		}
		compiled.WhenNodes = append(compiled.WhenNodes, whenNode)
	}
	if source.HasOtherwise {
		nodes, bindings, err := adaptNodeGroup(source.Otherwise, mapper)
		if err != nil {
			return nil, err
		}
		compiled.OtherwiseNode = &node.OtherwiseNode{Nodes: nodes, BindNodes: bindings}
	}
	return compiled, nil
}

func adaptNode(source configparser.Node, mapper *Mapper) (node.Node, error) {
	switch source := source.(type) {
	case configparser.TextNode:
		return adaptTextNode(source)
	case configparser.IfNode:
		return adaptIfNode(source, mapper)
	case configparser.ForeachNode:
		return adaptForeachNode(source, mapper)
	case configparser.ChooseNode:
		return adaptChooseNode(source, mapper)
	case configparser.TrimNode:
		return adaptTrimNode(source, mapper)
	case configparser.WhereNode:
		return adaptWhereNode(source, mapper)
	case configparser.SetNode:
		return adaptSetNode(source, mapper)
	case configparser.IncludeNode:
		return adaptIncludeNode(source, mapper)
	case configparser.BindNode:
		return nil, fmt.Errorf("bind node must be compiled as part of a node group")
	default:
		return nil, fmt.Errorf("unsupported parser node %T", source)
	}
}

func adaptMapper(mapper *Mapper, source configparser.Mapper) error {
	for _, fragment := range source.Fragments {
		nodes, bindNodes, err := adaptNodeGroup(fragment.Nodes, mapper)
		if err != nil {
			return err
		}
		if err := mapper.setSqlNode(&node.SQLNode{ID: fragment.ID, Nodes: nodes, BindNodes: bindNodes}); err != nil {
			return err
		}
	}

	for _, statementDocument := range source.Statements {
		if _, exists := mapper.statements[statementDocument.ID]; exists {
			return fmt.Errorf("duplicate statement id: %s", statementDocument.ID)
		}
		nodes, bindNodes, err := adaptNodeGroup(statementDocument.Nodes, mapper)
		if err != nil {
			return err
		}
		statement := &mappedStatement{
			mapper:    mapper,
			action:    juicesql.Action(statementDocument.Action),
			Nodes:     nodes,
			bindNodes: bindNodes,
			attrs:     maps.Clone(statementDocument.Attributes),
			id:        statementDocument.ID,
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
