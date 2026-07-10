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
	"io"

	"github.com/go-juicedev/juice/parser"
)

func (p *Parser) parseConfiguration(decoder *stdxml.Decoder) (*parser.Document, error) {
	document := &parser.Document{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("element <configuration> is not closed")
			}
			return nil, err
		}
		switch token := token.(type) {
		case stdxml.StartElement:
			switch token.Name.Local {
			case "settings":
				settings, err := parseSettings(decoder)
				if err != nil {
					return nil, err
				}
				document.Settings = settings
			case "environments":
				if p.IgnoreEnvironment {
					if err := skipElement(decoder, token); err != nil {
						return nil, err
					}
					continue
				}
				environments, err := parseEnvironments(decoder, token)
				if err != nil {
					return nil, err
				}
				document.Environments = environments
			case "mappers":
				if err := parseMappers(decoder, token, document); err != nil {
					return nil, err
				}
			default:
				return nil, wrap(token.Name.Local, fmt.Errorf("unknown configuration element"))
			}
		case stdxml.EndElement:
			if token.Name.Local == "configuration" {
				return document, nil
			}
		}
	}
}

func parseSettings(decoder *stdxml.Decoder) (map[string]string, error) {
	settings := make(map[string]string)
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch token := token.(type) {
		case stdxml.StartElement:
			if token.Name.Local != "setting" {
				return nil, wrap(token.Name.Local, fmt.Errorf("expected <setting>"))
			}
			name, err := requiredAttribute(token, "name")
			if err != nil {
				return nil, wrap("setting", err)
			}
			value, err := requiredAttribute(token, "value")
			if err != nil {
				return nil, wrap("setting", err)
			}
			if _, exists := settings[name]; exists {
				return nil, wrap("setting", fmt.Errorf("duplicate setting %q", name))
			}
			settings[name] = value
			if err := skipElement(decoder, token); err != nil {
				return nil, err
			}
		case stdxml.EndElement:
			if token.Name.Local == "settings" {
				return settings, nil
			}
		}
	}
}

func parseEnvironments(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Environments, error) {
	environments := parser.Environments{Default: attribute(start, "default"), Present: true}
	for {
		token, err := decoder.Token()
		if err != nil {
			return parser.Environments{}, err
		}
		switch token := token.(type) {
		case stdxml.StartElement:
			if token.Name.Local != "environment" {
				return parser.Environments{}, wrap(token.Name.Local, fmt.Errorf("expected <environment>"))
			}
			environment, err := parseEnvironment(decoder, token)
			if err != nil {
				return parser.Environments{}, err
			}
			environments.Items = append(environments.Items, environment)
		case stdxml.EndElement:
			if token.Name.Local == "environments" {
				return environments, nil
			}
		}
	}
}

func parseEnvironment(decoder *stdxml.Decoder, start stdxml.StartElement) (parser.Environment, error) {
	id, err := requiredAttribute(start, "id")
	if err != nil {
		return parser.Environment{}, wrap("environment", err)
	}
	environment := parser.Environment{ID: id, Attributes: attributes(start)}
	for {
		token, err := decoder.Token()
		if err != nil {
			return parser.Environment{}, err
		}
		switch token := token.(type) {
		case stdxml.StartElement:
			value, err := parseText(decoder, token.Name.Local)
			if err != nil {
				return parser.Environment{}, wrap(token.Name.Local, err)
			}
			switch token.Name.Local {
			case "driver":
				environment.Driver = value
			case "dataSource":
				environment.DataSource = value
			case "maxIdleConnNum":
				environment.MaxIdleConns = value
			case "maxOpenConnNum":
				environment.MaxOpenConns = value
			case "maxConnLifetime":
				environment.ConnMaxLifetime = value
			case "maxIdleConnLifetime":
				environment.ConnMaxIdleLifetime = value
			default:
				return parser.Environment{}, wrap(token.Name.Local, fmt.Errorf("unknown environment element"))
			}
		case stdxml.EndElement:
			if token.Name.Local == "environment" {
				return environment, nil
			}
		}
	}
}

func parseMappers(decoder *stdxml.Decoder, start stdxml.StartElement, document *parser.Document) error {
	document.MapperAttributes = attributes(start)
	if pattern := attribute(start, "pattern"); pattern != "" {
		source := parser.MapperSource{Pattern: pattern}
		document.MapperSources = append(document.MapperSources, source)
		document.MapperEntries = append(document.MapperEntries, parser.MapperEntry{Source: &source})
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch token := token.(type) {
		case stdxml.StartElement:
			if token.Name.Local != "mapper" {
				return wrap(token.Name.Local, fmt.Errorf("expected <mapper>"))
			}
			resource := attribute(token, "resource")
			mapperURL := attribute(token, "url")
			namespace := attribute(token, "namespace")
			set := 0
			for _, value := range []string{resource, mapperURL, namespace} {
				if value != "" {
					set++
				}
			}
			if set != 1 {
				return wrap("mapper", fmt.Errorf("exactly one of resource, url, or namespace is required"))
			}
			if resource != "" || mapperURL != "" {
				source := parser.MapperSource{Resource: resource, URL: mapperURL}
				document.MapperSources = append(document.MapperSources, source)
				document.MapperEntries = append(document.MapperEntries, parser.MapperEntry{Source: &source})
				if err := skipElement(decoder, token); err != nil {
					return err
				}
				continue
			}
			mapperDocument, err := parseMapper(decoder, token)
			if err != nil {
				return err
			}
			document.Mappers = append(document.Mappers, mapperDocument)
			document.MapperEntries = append(document.MapperEntries, parser.MapperEntry{Mapper: &mapperDocument})
		case stdxml.EndElement:
			if token.Name.Local == "mappers" {
				return nil
			}
		}
	}
}
