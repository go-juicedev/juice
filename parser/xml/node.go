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

	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
)

func parseNode(decoder *stdxml.Decoder, start stdxml.StartElement, resolver *includeResolver) (node.Node, error) {
	switch start.Name.Local {
	case "if":
		return parseIf(decoder, start, resolver)
	case "foreach":
		return parseForeach(decoder, start, resolver)
	case "choose":
		return parseChoose(decoder, resolver)
	case "trim":
		return parseTrim(decoder, start, resolver)
	case "where":
		return parseWhere(decoder, resolver)
	case "set":
		return parseSet(decoder, resolver)
	case "include":
		return parseInclude(decoder, start, resolver)
	default:
		return nil, wrap(start.Name.Local, fmt.Errorf("unknown dynamic SQL element"))
	}
}

func parseWhere(decoder *stdxml.Decoder, resolver *includeResolver) (node.Node, error) {
	var children group
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError("where", err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := strings.TrimSpace(string(token))
			if text != "" {
				children.nodes = append(children.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				children.binds = append(children.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return nil, err
			}
			children.nodes = append(children.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == "where" {
				return &WhereNode{Nodes: children.nodes, BindNodes: children.binds}, nil
			}
		}
	}
}

func parseSet(decoder *stdxml.Decoder, resolver *includeResolver) (node.Node, error) {
	var children group
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError("set", err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := strings.TrimSpace(string(token))
			if text != "" {
				children.nodes = append(children.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				children.binds = append(children.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return nil, err
			}
			children.nodes = append(children.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == "set" {
				return &SetNode{Nodes: children.nodes, BindNodes: children.binds}, nil
			}
		}
	}
}

func parseIf(decoder *stdxml.Decoder, start stdxml.StartElement, resolver *includeResolver) (node.Node, error) {
	test, err := requiredAttribute(start, "test")
	if err != nil {
		return nil, wrap("if", err)
	}
	var children group
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError("if", err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := strings.TrimSpace(string(token))
			if text != "" {
				children.nodes = append(children.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				children.binds = append(children.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return nil, err
			}
			children.nodes = append(children.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == "if" {
				compiled := &IfNode{Nodes: children.nodes, BindNodes: children.binds}
				if err := compiled.Parse(test); err != nil {
					return nil, err
				}
				return compiled, nil
			}
		}
	}
}

func parseBind(decoder *stdxml.Decoder, start stdxml.StartElement) (*bindNode, error) {
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
	return newBindNode(name, value)
}

func parseForeach(decoder *stdxml.Decoder, start stdxml.StartElement, resolver *includeResolver) (node.Node, error) {
	item, err := requiredAttribute(start, "item")
	if err != nil {
		return nil, wrap("foreach", err)
	}
	var children group
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError("foreach", err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := strings.TrimSpace(string(token))
			if text != "" {
				children.nodes = append(children.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				children.binds = append(children.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return nil, err
			}
			children.nodes = append(children.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == "foreach" {
				collection := attribute(start, "collection")
				if collection == "" {
					collection = eval.DefaultParamKey()
				}
				return &ForeachNode{
					Collection: collection,
					Item:       item,
					Index:      attribute(start, "index"),
					Open:       attribute(start, "open"),
					Close:      attribute(start, "close"),
					Separator:  attribute(start, "separator"),
					Nodes:      children.nodes,
					BindNodes:  children.binds,
				}, nil
			}
		}
	}
}

func parseChoose(decoder *stdxml.Decoder, resolver *includeResolver) (node.Node, error) {
	choose := &ChooseNode{}
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
				choose.BindNodes = append(choose.BindNodes, parsed)
			case "when":
				when, err := parseWhen(decoder, token, resolver)
				if err != nil {
					return nil, err
				}
				choose.WhenNodes = append(choose.WhenNodes, when)
			case "otherwise":
				if choose.OtherwiseNode != nil {
					return nil, wrap("otherwise", fmt.Errorf("element may only appear once"))
				}
				otherwise, err := parseOtherwise(decoder, resolver)
				if err != nil {
					return nil, err
				}
				choose.OtherwiseNode = otherwise
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

func parseWhen(decoder *stdxml.Decoder, start stdxml.StartElement, resolver *includeResolver) (node.Node, error) {
	test, err := requiredAttribute(start, "test")
	if err != nil {
		return nil, wrap("when", err)
	}
	var children group
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError("when", err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := strings.TrimSpace(string(token))
			if text != "" {
				children.nodes = append(children.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				children.binds = append(children.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return nil, err
			}
			children.nodes = append(children.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == "when" {
				when := &WhenNode{
					Nodes:     children.nodes,
					BindNodes: children.binds,
				}
				if err := when.Parse(test); err != nil {
					return nil, err
				}
				return when, nil
			}
		}
	}
}

func parseOtherwise(decoder *stdxml.Decoder, resolver *includeResolver) (node.Node, error) {
	var children group
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError("otherwise", err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := strings.TrimSpace(string(token))
			if text != "" {
				children.nodes = append(children.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				children.binds = append(children.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return nil, err
			}
			children.nodes = append(children.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == "otherwise" {
				return &OtherwiseNode{
					Nodes:     children.nodes,
					BindNodes: children.binds,
				}, nil
			}
		}
	}
}

func parseTrim(decoder *stdxml.Decoder, start stdxml.StartElement, resolver *includeResolver) (node.Node, error) {
	var children group
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, elementReadError("trim", err)
		}
		switch token := token.(type) {
		case stdxml.CharData:
			text := strings.TrimSpace(string(token))
			if text != "" {
				children.nodes = append(children.nodes, NewTextNode(text))
			}
		case stdxml.StartElement:
			if token.Name.Local == "bind" {
				binding, err := parseBind(decoder, token)
				if err != nil {
					return nil, err
				}
				children.binds = append(children.binds, binding)
				continue
			}
			n, err := parseNode(decoder, token, resolver)
			if err != nil {
				return nil, err
			}
			children.nodes = append(children.nodes, n)
		case stdxml.EndElement:
			if token.Name.Local == "trim" {
				return &TrimNode{
					Prefix:          attribute(start, "prefix"),
					Suffix:          attribute(start, "suffix"),
					PrefixOverrides: splitOverrides(attribute(start, "prefixOverrides")),
					SuffixOverrides: splitOverrides(attribute(start, "suffixOverrides")),
					Nodes:           children.nodes,
					BindNodes:       children.binds,
				}, nil
			}
		}
	}
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

func parseInclude(decoder *stdxml.Decoder, start stdxml.StartElement, resolver *includeResolver) (node.Node, error) {
	refID, err := requiredAttribute(start, "refid")
	if err != nil {
		return nil, wrap("include", err)
	}
	properties := make(eval.H)
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
			if _, exists := properties[name]; exists {
				return nil, wrap("property", fmt.Errorf("duplicate property %q", name))
			}
			properties[name] = value
			if err := skipElement(decoder, token); err != nil {
				return nil, err
			}
		case stdxml.EndElement:
			if token.Name.Local == "include" {
				include := newIncludeNode(refID, resolver)
				if len(properties) > 0 {
					include.WithProperties(properties)
				}
				return include, nil
			}
		}
	}
}
