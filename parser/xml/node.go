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

func parseNodes(decoder *stdxml.Decoder, end string, preserveWhitespace bool) ([]parser.Node, error) {
	var nodes []parser.Node
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError(end, err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := string(token)
			if strings.TrimSpace(text) == "" {
				continue
			}
			if !preserveWhitespace {
				text = strings.TrimSpace(text)
			}
			nodes = append(nodes, parser.TextNode{Text: text})
		case stdxml.StartElement:
			node, err := parseNode(decoder, token)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
		case stdxml.EndElement:
			if token.Name.Local == end {
				return nodes, nil
			}
		}
	}
}

func parseNode(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Node, error) {
	switch start.Name.Local {
	case "if":
		return parseIf(decoder, start)
	case "bind":
		return parseBind(decoder, start)
	case "foreach":
		return parseForeach(decoder, start)
	case "choose":
		return parseChoose(decoder)
	case "trim":
		return parseTrim(decoder, start)
	case "where":
		children, err := parseNodes(decoder, "where", false)
		return parser.WhereNode{Children: children}, err
	case "set":
		children, err := parseNodes(decoder, "set", false)
		return parser.SetNode{Children: children}, err
	case "include":
		return parseInclude(decoder, start)
	default:
		return nil, wrap(start.Name.Local, fmt.Errorf("unknown dynamic SQL element"))
	}
}

func parseIf(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Node, error) {
	test, err := requiredAttribute(start, "test")
	if err != nil {
		return nil, wrap("if", err)
	}
	children, err := parseNodes(decoder, "if", false)
	if err != nil {
		return nil, err
	}
	return parser.IfNode{Test: test, Children: children}, nil
}

func parseBind(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Node, error) {
	name, err := requiredAttribute(start, "name")
	if err != nil {
		return nil, wrap("bind", err)
	}
	value, err := requiredAttribute(start, "value")
	if err != nil {
		return nil, wrap("bind", err)
	}
	if err := skipElement(decoder, start); err != nil {
		return nil, err
	}
	return parser.BindNode{Name: name, Value: value}, nil
}

func parseForeach(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Node, error) {
	item, err := requiredAttribute(start, "item")
	if err != nil {
		return nil, wrap("foreach", err)
	}
	children, err := parseNodes(decoder, "foreach", false)
	if err != nil {
		return nil, err
	}
	return parser.ForeachNode{
		Collection: attribute(start, "collection"),
		Item:       item,
		Index:      attribute(start, "index"),
		Open:       attribute(start, "open"),
		Close:      attribute(start, "close"),
		Separator:  attribute(start, "separator"),
		Children:   children,
	}, nil
}

func parseChoose(decoder *stdxml.Decoder) (parser.Node, error) {
	choose := parser.ChooseNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch token := token.(type) {
		case stdxml.CharData:
			if strings.TrimSpace(string(token)) != "" {
				return nil, wrap("choose", fmt.Errorf("text is not allowed directly inside choose"))
			}
		case stdxml.StartElement:
			switch token.Name.Local {
			case "bind":
				parsed, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				binding := parsed.(parser.BindNode)
				choose.Bindings = append(choose.Bindings, binding)
			case "when":
				test, err := requiredAttribute(token, "test")
				if err != nil {
					return nil, wrap("when", err)
				}
				children, err := parseNodes(decoder, "when", false)
				if err != nil {
					return nil, err
				}
				choose.Whens = append(choose.Whens, parser.WhenNode{Test: test, Children: children})
			case "otherwise":
				if choose.HasOtherwise {
					return nil, wrap("otherwise", fmt.Errorf("element may only appear once"))
				}
				children, err := parseNodes(decoder, "otherwise", false)
				if err != nil {
					return nil, err
				}
				choose.Otherwise = children
				choose.HasOtherwise = true
			default:
				return nil, wrap(token.Name.Local, fmt.Errorf("expected <when> or <otherwise>"))
			}
		case stdxml.EndElement:
			if token.Name.Local == "choose" {
				return choose, nil
			}
		}
	}
}

func parseTrim(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Node, error) {
	children, err := parseNodes(decoder, "trim", false)
	if err != nil {
		return nil, err
	}
	return parser.TrimNode{
		Prefix:          attribute(start, "prefix"),
		Suffix:          attribute(start, "suffix"),
		PrefixOverrides: attribute(start, "prefixOverrides"),
		SuffixOverrides: attribute(start, "suffixOverrides"),
		Children:        children,
	}, nil
}

func parseInclude(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Node, error) {
	refID, err := requiredAttribute(start, "refid")
	if err != nil {
		return nil, wrap("include", err)
	}
	include := parser.IncludeNode{RefID: refID}
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch token := token.(type) {
		case stdxml.CharData:
			if strings.TrimSpace(string(token)) != "" {
				return nil, wrap("include", fmt.Errorf("text is not allowed inside include"))
			}
		case stdxml.StartElement:
			if token.Name.Local != "property" {
				return nil, wrap(token.Name.Local, fmt.Errorf("expected <property>"))
			}
			name, err := requiredAttribute(token, "name")
			if err != nil {
				return nil, wrap("property", err)
			}
			value, err := requiredAttribute(token, "value")
			if err != nil {
				return nil, wrap("property", err)
			}
			if include.Properties == nil {
				include.Properties = make(map[string]string)
			}
			if _, exists := include.Properties[name]; exists {
				return nil, wrap("property", fmt.Errorf("duplicate property %q", name))
			}
			include.Properties[name] = value
			if err := skipElement(decoder, token); err != nil {
				return nil, err
			}
		case stdxml.EndElement:
			if token.Name.Local == "include" {
				return include, nil
			}
		}
	}
}
