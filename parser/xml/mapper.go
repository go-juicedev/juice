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

	state := mapperParseState{
		decoder:      decoder,
		document:     parser.Mapper{Namespace: namespace},
		resolver:     &includeResolver{namespace: namespace, registry: registry},
		registry:     registry,
		statementIDs: make(map[string]struct{}),
	}
	return state.parse()
}

type mapperParseState struct {
	decoder      *stdxml.Decoder
	document     parser.Mapper
	resolver     *includeResolver
	registry     *sqlRegistry
	statementIDs map[string]struct{}
}

func (s *mapperParseState) parse() (parser.Mapper, error) {
	for {
		token, err := s.decoder.Token()
		if err != nil {
			return parser.Mapper{}, elementReadError("mapper", err)
		}
		switch token := token.(type) {
		case stdxml.StartElement:
			if err := s.parseElement(token); err != nil {
				return parser.Mapper{}, err
			}
		case stdxml.EndElement:
			if token.Name.Local == "mapper" {
				return s.document, nil
			}
		}
	}
}

func (s *mapperParseState) parseElement(start stdxml.StartElement) error {
	action := parser.Action(start.Name.Local)
	switch action {
	case parser.Select, parser.Insert, parser.Update, parser.Delete:
		return s.parseStatementElement(start, action)
	case "sql":
		return s.parseSQLElement(start)
	default:
		return wrap(start.Name.Local, fmt.Errorf("unknown mapper element"))
	}
}

func (s *mapperParseState) parseStatementElement(start stdxml.StartElement, action parser.Action) error {
	statement, err := parseStatement(s.decoder, start, action, s.resolver)
	if err != nil {
		return err
	}
	if _, exists := s.statementIDs[statement.ID]; exists {
		return wrap(start.Name.Local, fmt.Errorf("duplicate statement id %q", statement.ID))
	}
	s.statementIDs[statement.ID] = struct{}{}
	s.document.Statements = append(s.document.Statements, statement)
	return nil
}

func (s *mapperParseState) parseSQLElement(start stdxml.StartElement) error {
	sql, err := parseSQL(s.decoder, start, s.resolver)
	if err != nil {
		return err
	}
	if err := s.registry.register(s.document.Namespace+"."+sql.ID(), sql); err != nil {
		return wrap("sql", err)
	}
	return nil
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
	sqlResolver := *resolver
	sqlResolver.owner = resolver.namespace + "." + id
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
			n, err := parseNode(decoder, token, &sqlResolver)
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
