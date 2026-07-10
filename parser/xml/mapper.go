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

	"github.com/go-juicedev/juice/parser"
)

func parseMapper(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Mapper, error) {
	namespace, err := requiredAttribute(start, "namespace")
	if err != nil {
		return parser.Mapper{}, wrap("mapper", err)
	}
	mapperDocument := parser.Mapper{
		Namespace:  namespace,
		Attributes: attributes(start),
	}
	statementIDs := make(map[string]struct{})
	fragmentIDs := make(map[string]struct{})

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
				statement, err := parseStatement(decoder, token, action)
				if err != nil {
					return parser.Mapper{}, err
				}
				if _, exists := statementIDs[statement.ID]; exists {
					return parser.Mapper{}, wrap(token.Name.Local, fmt.Errorf("duplicate statement id %q", statement.ID))
				}
				statementIDs[statement.ID] = struct{}{}
				mapperDocument.Statements = append(mapperDocument.Statements, statement)
			case "sql":
				fragment, err := parseFragment(decoder, token)
				if err != nil {
					return parser.Mapper{}, err
				}
				if _, exists := fragmentIDs[fragment.ID]; exists {
					return parser.Mapper{}, wrap("sql", fmt.Errorf("duplicate fragment id %q", fragment.ID))
				}
				fragmentIDs[fragment.ID] = struct{}{}
				mapperDocument.Fragments = append(mapperDocument.Fragments, fragment)
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

func parseStatement(decoder *stdxml.Decoder, start stdxml.StartElement, action parser.Action) (parser.Statement, error) {
	id, err := requiredAttribute(start, "id")
	if err != nil {
		return parser.Statement{}, wrap(start.Name.Local, err)
	}
	nodes, err := parseNodes(decoder, start.Name.Local, true)
	if err != nil {
		return parser.Statement{}, err
	}
	return parser.Statement{
		ID:         id,
		Action:     action,
		Attributes: attributes(start),
		Nodes:      nodes,
	}, nil
}

func parseFragment(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Fragment, error) {
	id, err := requiredAttribute(start, "id")
	if err != nil {
		return parser.Fragment{}, wrap("sql", err)
	}
	nodes, err := parseNodes(decoder, "sql", false)
	if err != nil {
		return parser.Fragment{}, err
	}
	return parser.Fragment{ID: id, Nodes: nodes}, nil
}
