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

package xml

import (
	stdxml "encoding/xml"
	"fmt"
	"strings"

	"github.com/go-juicedev/juice/parser"
)

func parseMapper(decoder *stdxml.Decoder, start stdxml.StartElement, registry *sqlRegistry) (parser.Mapper, error) {
	if err := validateAttributes(start, "namespace"); err != nil {
		return parser.Mapper{}, wrap("mapper", err)
	}
	namespace, err := requiredAttribute(start, "namespace")
	if err != nil {
		return parser.Mapper{}, wrap("mapper", err)
	}
	mapperDocument := parser.Mapper{Namespace: namespace}
	resolver := &includeResolver{namespace: namespace, registry: registry}
	statementIDs := make(map[string]struct{})
	sqlIDs := make(map[string]struct{})

	for {
		token, err := decoder.Token()
		if err != nil {
			return parser.Mapper{}, elementReadError("mapper", err)
		}
		switch token := token.(type) {
		case stdxml.StartElement:
			action := parser.Action(token.Name.Local)
			switch action {
			case parser.Select, parser.Insert, parser.Update, parser.Delete:
				statement, err := parseStatement(decoder, token, action, resolver)
				if err != nil {
					return parser.Mapper{}, err
				}
				if _, exists := statementIDs[statement.ID]; exists {
					return parser.Mapper{}, wrap(token.Name.Local, fmt.Errorf("duplicate statement id %q", statement.ID))
				}
				statementIDs[statement.ID] = struct{}{}
				mapperDocument.Statements = append(mapperDocument.Statements, statement)
			case "sql":
				sql, err := parseSQL(decoder, token, resolver)
				if err != nil {
					return parser.Mapper{}, err
				}
				if _, exists := sqlIDs[sql.ID()]; exists {
					return parser.Mapper{}, wrap("sql", fmt.Errorf("duplicate sql id %q", sql.ID()))
				}
				sqlIDs[sql.ID()] = struct{}{}
				if err := registry.register(namespace+"."+sql.ID(), sql); err != nil {
					return parser.Mapper{}, wrap("sql", err)
				}
			default:
				return parser.Mapper{}, wrap(token.Name.Local, fmt.Errorf("unknown mapper element"))
			}
		case stdxml.EndElement:
			if token.Name.Local == "mapper" {
				return mapperDocument, nil
			}
		}
	}
}

func parseStatement(decoder *stdxml.Decoder, start stdxml.StartElement, action parser.Action, resolver *includeResolver) (parser.Statement, error) {
	id, err := requiredAttribute(start, "id")
	if err != nil {
		return parser.Statement{}, wrap(start.Name.Local, err)
	}
	var nodes group
	for {
		token, err := decoder.Token()
		if err != nil {
			return parser.Statement{}, elementReadError(start.Name.Local, err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := string(token)
			if strings.TrimSpace(text) != "" {
				nodes.nodes = append(nodes.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return parser.Statement{}, err
				}
				nodes.binds = append(nodes.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return parser.Statement{}, err
			}
			nodes.nodes = append(nodes.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == start.Name.Local {
				return parser.Statement{
					ID:         id,
					Action:     action,
					Attributes: attributes(start),
					Node:       nodes,
				}, nil
			}
		}
	}
}

func parseSQL(decoder *stdxml.Decoder, start stdxml.StartElement, resolver *includeResolver) (*SQL, error) {
	id, err := requiredAttribute(start, "id")
	if err != nil {
		return nil, wrap("sql", err)
	}
	var nodes group
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError("sql", err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := strings.TrimSpace(string(token))
			if text != "" {
				nodes.nodes = append(nodes.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				nodes.binds = append(nodes.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return nil, err
			}
			nodes.nodes = append(nodes.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == "sql" {
				return &SQL{id: id, node: nodes}, nil
			}
		}
	}
}
