/*
Copyright 2023 eatmoreapple

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
	"encoding/xml"
	"errors"
	"fmt"
	gotoken "go/token"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-juicedev/juice/eval"
)

// ConfigurationParser is the interface for parsing configuration.
type ConfigurationParser interface {
	// Parse parses the configuration from the reader.
	Parse(reader io.Reader) (IConfiguration, error)
}

// XMLParser is the parser for XML configuration.
type XMLParser struct {
	configuration Configuration
	FS            fs.FS
	ignoreEnv     bool
	parsers       []XMLElementParser
}

// Parse implements ConfigurationParser.
func (p *XMLParser) Parse(reader io.Reader) (IConfiguration, error) {
	parserChain := XMLElementParserChain(p.parsers)
	decoder := xml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		startElement, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if err = parserChain.ParseElement(p, decoder, startElement); err != nil {
			if !errors.Is(err, errNoXMLElementMatched) {
				return nil, err
			}
		}
	}
	return &p.configuration, nil
}

func (p *XMLParser) AddXMLElementParser(parsers ...XMLElementParser) {
	p.parsers = append(p.parsers, parsers...)
}

type XMLElementParser interface {
	ParseElement(parser *XMLParser, decoder *xml.Decoder, token xml.StartElement) error
	MatchElement(token xml.StartElement) bool
}

// errNoXMLElementMatched is an error that indicates no XML element matched the expected criteria.
var errNoXMLElementMatched = errors.New("no xml element matched")

type XMLElementParserChain []XMLElementParser

func (xs XMLElementParserChain) ParseElement(parser *XMLParser, decoder *xml.Decoder, token xml.StartElement) error {
	for _, x := range xs {
		if x.MatchElement(token) {
			return x.ParseElement(parser, decoder, token)
		}
	}
	return errNoXMLElementMatched
}

type XMLEnvironmentsElementParser struct{}

func (p *XMLEnvironmentsElementParser) MatchElement(token xml.StartElement) bool {
	return token.Name.Local == "environments"
}

func (p *XMLEnvironmentsElementParser) ParseElement(parser *XMLParser, decoder *xml.Decoder, token xml.StartElement) error {
	if parser.ignoreEnv {
		return nil
	}
	envs, err := p.parseEnvironments(decoder, token)
	if err != nil {
		return err
	}
	parser.configuration.environments = envs
	return err
}

func (p *XMLEnvironmentsElementParser) parseEnvironment(decoder *xml.Decoder, token xml.StartElement) (*Environment, error) {
	var env = &Environment{}
	for _, attr := range token.Attr {
		env.setAttr(attr.Name.Local, attr.Value)
	}
	id := env.ID()
	if id != "" {
		if !gotoken.IsIdentifier(id) {
			return nil, fmt.Errorf("environment id is invalid: %s", id)
		}
	} else {
		return nil, errors.New("environment id is required")
	}
	provider := env.provider()
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			tokenName := token.Name.Local
			switch tokenName {
			case "dataSource":
				env.DataSource, err = parseString(tokenName, decoder, provider)
				if err != nil {
					return nil, err
				}
			case "driver":
				env.Driver, err = parseString(tokenName, decoder, provider)
				if err != nil {
					return nil, err
				}
			case "maxIdleConnNum":
				env.MaxIdleConnNum, err = parseInt(tokenName, decoder, provider)
				if err != nil {
					return nil, err
				}
			case "maxOpenConnNum":
				env.MaxOpenConnNum, err = parseInt(tokenName, decoder, provider)
				if err != nil {
					return nil, err
				}
			case "maxConnLifetime":
				env.MaxConnLifetime, err = parseInt(tokenName, decoder, provider)
				if err != nil {
					return nil, err
				}
			case "maxIdleConnLifetime":
				env.MaxIdleConnLifetime, err = parseInt(tokenName, decoder, provider)
				if err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if token.Name.Local == "environment" {
				return env, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "environment"}
}

func (p *XMLEnvironmentsElementParser) parseEnvironments(decoder *xml.Decoder, token xml.StartElement) (*environments, error) {
	var envs environments
	for _, attr := range token.Attr {
		envs.setAttr(attr.Name.Local, attr.Value)
	}
	if envs.Attribute("default") == "" {
		return nil, errors.New("default environment is not specified")
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "environment" {
				environment, err := p.parseEnvironment(decoder, token)
				if err != nil {
					return nil, err
				}
				if _, exists := envs.envs[environment.ID()]; exists {
					return nil, fmt.Errorf("duplicate environment id: %s", environment.ID())
				}
				if envs.envs == nil {
					envs.envs = make(map[string]*Environment)
				}
				envs.envs[environment.ID()] = environment
			}
		case xml.EndElement:
			if token.Name.Local == "environments" {
				return &envs, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "environments"}
}

type XMLSettingsElementParser struct{}

func (p *XMLSettingsElementParser) MatchElement(token xml.StartElement) bool {
	return token.Name.Local == "settings"
}

func (p *XMLSettingsElementParser) ParseElement(parser *XMLParser, decoder *xml.Decoder, token xml.StartElement) error {
	settings, err := p.parseSettings(decoder)
	if err != nil {
		return err
	}
	parser.configuration.settings = settings
	return nil
}

func (p *XMLSettingsElementParser) parseSettings(decoder *xml.Decoder) (keyValueSettingProvider, error) {
	var settings = make(keyValueSettingProvider)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		switch t := token.(type) {
		case xml.EndElement:
			if t.Name.Local == "settings" {
				return settings, nil
			}
		case xml.StartElement:
			if t.Name.Local != "setting" {
				continue
			}
			var item settingItem
			if err := decoder.DecodeElement(&item, &t); err != nil {
				return nil, err
			}
			if _, ok := settings[item.Name]; ok {
				return nil, fmt.Errorf("duplicate setting name: %s", item.Name)
			}
			settings[item.Name] = item.Value
		}
	}

	return settings, nil
}

type XMLMappersElementParser struct {
	parser *XMLParser
}

func (p *XMLMappersElementParser) MatchElement(token xml.StartElement) bool {
	return token.Name.Local == "mappers"
}

func (p *XMLMappersElementParser) ParseElement(parser *XMLParser, decoder *xml.Decoder, token xml.StartElement) error {
	p.parser = parser
	mappers, err := p.parseMappers(token, decoder)
	if err != nil {
		return err
	}
	mappers.cfg = parser.configuration
	parser.configuration.mappers = mappers
	return nil
}

func (p *XMLMappersElementParser) parseMappers(start xml.StartElement, decoder *xml.Decoder) (*Mappers, error) {
	var mappers = &Mappers{}

	for _, attr := range start.Attr {
		mappers.setAttribute(attr.Name.Local, attr.Value)
	}

	// parse mappers by pattern
	if pattern := mappers.Attribute("pattern"); pattern != "" {
		matchedMappers, err := p.parseMapperByPattern(pattern)
		if err != nil {
			return nil, err
		}
		for _, mapper := range matchedMappers {
			if err = mappers.setMapper(mapper.namespace, mapper); err != nil {
				return nil, err
			}
		}
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			// mappers only support mapper child node
			if token.Name.Local == "mapper" {
				mapper, err := p.parseMapper(decoder, token)
				if err != nil {
					return nil, err
				}
				if err = mappers.setMapper(mapper.namespace, mapper); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if token.Name.Local == "mappers" {
				return mappers, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "mappers"}
}

func (p *XMLMappersElementParser) parseMapper(decoder *xml.Decoder, token xml.StartElement) (*Mapper, error) {
	mapper := &Mapper{}
	for _, attr := range token.Attr {
		mapper.setAttribute(attr.Name.Local, attr.Value)
	}

	resource := mapper.Attribute("resource")
	_url := mapper.Attribute("url")
	namespace := mapper.Attribute("namespace")

	// check conflict
	// resource, url, namespace only one can be set
	// namespace is required if resource and url are not set
	switch {
	case resource != "" && _url != "":
		return nil, &nodeAttributeConflictError{nodeName: "mapper", attrName: "resource|url"}
	case resource != "" && namespace != "":
		return nil, &nodeAttributeConflictError{nodeName: "mapper", attrName: "resource|namespace"}
	case _url != "" && namespace != "":
		return nil, &nodeAttributeConflictError{nodeName: "mapper", attrName: "url|namespace"}
	case resource == "" && _url == "" && namespace == "":
		return nil, &nodeAttributeRequiredError{nodeName: "mapper", attrName: "resource|url|namespace"}
	}
	if resource != "" {
		return p.parseMapperByResource(resource)
	}
	if _url != "" {
		return p.parseMapperByURL(_url)
	}
	if namespace == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "mapper", attrName: "namespace"}
	}

	mapper.namespace = namespace
	mapper.statements = make(map[string]*xmlSQLStatement)

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			action := action(token.Name.Local)
			switch action {
			case Select, Insert, Update, Delete:
				stmt := &xmlSQLStatement{action: action, mapper: mapper}
				if err = p.parseStatement(stmt, decoder, token); err != nil {
					return nil, err
				}
				key := stmt.ID()
				if _, exists := mapper.statements[key]; exists {
					return nil, fmt.Errorf("duplicate xmlSQLStatement id: %s", key)
				}
				mapper.statements[key] = stmt
			case "sql":
				// parse sql node
				sqlNode, err := p.parseSQLNode(mapper, decoder, token)
				if err != nil {
					return nil, err
				}
				if err = mapper.setSqlNode(sqlNode); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if token.Name.Local == "mapper" {
				return mapper, nil
			}
		}
	}
	return mapper, nil
}

func (p *XMLMappersElementParser) parseMapperByReader(reader io.Reader) (mapper *Mapper, err error) {
	decoder := xml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "mapper" {
				if mapper, err = p.parseMapper(decoder, token); err != nil {
					return nil, err
				}
				break
			}
		}
	}
	return mapper, err
}

func (p *XMLMappersElementParser) parseMapperByResource(resource string) (*Mapper, error) {
	var (
		reader io.ReadCloser
		err    error
	)
	reader, err = p.parser.FS.Open(resource)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return p.parseMapperByReader(reader)
}

func (p *XMLMappersElementParser) parseMapperByHttpResponse(url string) (*Mapper, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return p.parseMapperByReader(resp.Body)
}

func (p *XMLMappersElementParser) parseMapperByURL(path string) (*Mapper, error) {
	// prepare url schema
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	schema := u.Scheme
	switch schema {
	case "file":
		return p.parseMapperByResource(u.Path)
	case "http", "https":
		return p.parseMapperByHttpResponse(path)
	default:
		return nil, errors.New("invalid url schema")
	}
}

func (p *XMLMappersElementParser) parseMapperByPattern(pattern string) ([]*Mapper, error) {
	fsys := p.parser.FS
	// Find files matching the pattern using fs.Glob
	matches, err := fs.Glob(fsys, pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}

	if len(matches) == 0 {
		return nil, nil
	}

	handler := func(match string) (*Mapper, error) {
		// Open and read the file
		file, err := fsys.Open(match)
		if err != nil {
			return nil, fmt.Errorf("failed to open mapper file %q: %w", match, err)
		}
		defer func() { _ = file.Close() }()

		// Parse mapper from file content
		mapper, err := p.parseMapperByReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mapper %q: %w", match, err)
		}

		return mapper, nil
	}

	// Pre-allocate slice with capacity matching number of files
	mappers := make([]*Mapper, 0, len(matches))

	// Process each matched file
	for _, match := range matches {
		// Parse mapper from file content
		mapper, err := handler(match)
		if err != nil {
			return nil, err
		}

		mappers = append(mappers, mapper)
	}

	return mappers, nil
}

func (p *XMLMappersElementParser) parseStatement(stmt *xmlSQLStatement, decoder *xml.Decoder, token xml.StartElement) error {
	for _, attr := range token.Attr {
		stmt.setAttribute(attr.Name.Local, attr.Value)
	}
	if id := stmt.Attribute("id"); id == "" {
		return fmt.Errorf("%s xmlSQLStatement id is required", stmt.Action())
	} else {
		stmt.id = id
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(stmt.mapper, decoder, token)
			if err != nil {
				return err
			}
			stmt.Nodes = append(stmt.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := NewTextNode(char)
				stmt.Nodes = append(stmt.Nodes, node)
			}
		case xml.EndElement:
			switch token.Name.Local {
			case stmt.action.String():
				return nil
			default:
				return fmt.Errorf("unexpected end element: %s", token.Name.Local)
			}
		}
	}
	return nil
}

func (p *XMLMappersElementParser) parseTags(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	switch token.Name.Local {
	case "if":
		return p.parseIf(mapper, decoder, token)
	case "where":
		return p.parseWhere(mapper, decoder)
	case "trim":
		return p.parseTrim(mapper, decoder, token)
	case "foreach":
		return p.parseForeach(mapper, decoder, token)
	case "set":
		return p.parseSet(mapper, decoder)
	case "include":
		return p.parseInclude(mapper, decoder, token)
	case "choose":
		return p.parseChoose(mapper, decoder)
	}
	return nil, fmt.Errorf("unknown tag: %s", token.Name.Local)
}

func (p *XMLMappersElementParser) parseInclude(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	var ref string
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "refid":
			ref = attr.Value
		}
	}
	if ref == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "include", attrName: "refid"}
	}

	// try to find sql xmlSQLStatement by refid
	// if not found, it means the sql xmlSQLStatement has not been parsed yet,
	// ignore it and lazy parse it when use
	sqlNode, _ := mapper.GetSQLNodeByID(ref)

	includeNode := &IncludeNode{sqlNode: sqlNode, mapper: mapper, refId: ref}

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		// TODO: PARSE PROPERTIES HERE
		case xml.EndElement:
			if token.Name.Local == "include" {
				return includeNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "include"}
}

func (p *XMLMappersElementParser) parseSet(mapper *Mapper, decoder *xml.Decoder) (Node, error) {
	setNode := &SetNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			setNode.Nodes = append(setNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := NewTextNode(char)
				setNode.Nodes = append(setNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "set" {
				return setNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "set"}
}

func (p *XMLMappersElementParser) parseIf(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	ifNode := &IfNode{}
	var test string
	for _, attr := range token.Attr {
		if attr.Name.Local == "test" {
			test = attr.Value
			break
		}
	}
	if test == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "if", attrName: "test"}
	}

	// parse condition expression
	if err := ifNode.Parse(test); err != nil {
		return nil, err
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			ifNode.Nodes = append(ifNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := NewTextNode(char)
				ifNode.Nodes = append(ifNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "if" {
				return ifNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "if"}
}

func (p *XMLMappersElementParser) parseWhere(mapper *Mapper, decoder *xml.Decoder) (Node, error) {
	whereNode := &WhereNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			whereNode.Nodes = append(whereNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := NewTextNode(char)
				whereNode.Nodes = append(whereNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "where" {
				return whereNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "where"}
}

func (p *XMLMappersElementParser) parseTrim(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	trimNode := &TrimNode{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "prefix":
			trimNode.Prefix = attr.Value
		case "prefixOverrides":
			prefixOverrides := strings.Split(attr.Value, "|")
			for i := range prefixOverrides {
				prefixOverrides[i] = strings.TrimSpace(prefixOverrides[i])
			}
			trimNode.PrefixOverrides = prefixOverrides
		case "suffix":
			trimNode.Suffix = attr.Value
		case "suffixOverrides":
			suffixOverrides := strings.Split(attr.Value, "|")
			for i := range suffixOverrides {
				suffixOverrides[i] = strings.TrimSpace(suffixOverrides[i])
			}
			trimNode.SuffixOverrides = suffixOverrides
		}
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			trimNode.Nodes = append(trimNode.Nodes, node)
		case xml.EndElement:
			if token.Name.Local == "trim" {
				return trimNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "trim"}
}

func (p *XMLMappersElementParser) parseForeach(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	foreachNode := &ForeachNode{}
	for _, attr := range token.Attr {
		switch attr.Name.Local {
		case "collection":
			foreachNode.Collection = attr.Value
		case "item":
			foreachNode.Item = attr.Value
		case "index":
			foreachNode.Index = attr.Value
		case "open":
			foreachNode.Open = attr.Value
		case "separator":
			foreachNode.Separator = attr.Value
		case "close":
			foreachNode.Close = attr.Value
		}
	}

	// if collection is empty, use default param key instead.
	if foreachNode.Collection == "" {
		foreachNode.Collection = eval.DefaultParamKey()
	}
	if foreachNode.Item == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "foreach", attrName: "item"}
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			foreachNode.Nodes = append(foreachNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := NewTextNode(char)
				foreachNode.Nodes = append(foreachNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "foreach" {
				return foreachNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "foreach"}
}

func (p *XMLMappersElementParser) parseChoose(mapper *Mapper, decoder *xml.Decoder) (Node, error) {
	chooseNode := &ChooseNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "when":
				node, err := p.parseWhen(mapper, decoder, token)
				if err != nil {
					return nil, err
				}
				chooseNode.WhenNodes = append(chooseNode.WhenNodes, node)
			case "otherwise":
				if chooseNode.OtherwiseNode != nil {
					return nil, errors.New("otherwise is only once")
				}
				node, err := p.parseOtherwise(mapper, decoder)
				if err != nil {
					return nil, err
				}
				chooseNode.OtherwiseNode = node
			}

		case xml.EndElement:
			if token.Name.Local == "choose" {
				return chooseNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "choose"}
}

func (p *XMLMappersElementParser) parseSQLNode(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (*SQLNode, error) {
	var sqlNode = &SQLNode{}
	for _, attr := range token.Attr {
		if attr.Name.Local == "id" {
			sqlNode.id = attr.Value
			break
		}
	}
	if sqlNode.id == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "sql", attrName: "id"}
	}
	if strings.Contains(sqlNode.id, ".") {
		return nil, fmt.Errorf("sql id can not contain '.' %s", sqlNode.id)
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			tags, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			sqlNode.nodes = append(sqlNode.nodes, tags)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := NewTextNode(char)
				sqlNode.nodes = append(sqlNode.nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "sql" {
				return sqlNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "sql"}
}

func (p *XMLMappersElementParser) parseWhen(mapper *Mapper, decoder *xml.Decoder, token xml.StartElement) (Node, error) {
	whenNode := &WhenNode{}
	var test string
	for _, attr := range token.Attr {
		if attr.Name.Local == "test" {
			test = attr.Value
			break
		}
	}
	if test == "" {
		return nil, &nodeAttributeRequiredError{nodeName: "when", attrName: "test"}
	}

	// parse condition expression
	if err := whenNode.Parse(test); err != nil {
		return nil, err
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			node, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			whenNode.Nodes = append(whenNode.Nodes, node)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := NewTextNode(char)
				whenNode.Nodes = append(whenNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "when" {
				return whenNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "when"}
}

func (p *XMLMappersElementParser) parseOtherwise(mapper *Mapper, decoder *xml.Decoder) (Node, error) {
	otherwiseNode := &OtherwiseNode{}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			tags, err := p.parseTags(mapper, decoder, token)
			if err != nil {
				return nil, err
			}
			otherwiseNode.Nodes = append(otherwiseNode.Nodes, tags)
		case xml.CharData:
			text := string(token)
			if char := strings.TrimSpace(text); char != "" {
				node := NewTextNode(char)
				otherwiseNode.Nodes = append(otherwiseNode.Nodes, node)
			}
		case xml.EndElement:
			if token.Name.Local == "otherwise" {
				return otherwiseNode, nil
			}
		}
	}
	return nil, &nodeUnclosedError{nodeName: "otherwise"}
}

// parseCharData reads character data from an XML decoder until it encounters the specified end element.
// It returns the character data as a string or an error if one occurs.
func parseCharData(decoder *xml.Decoder, endElementName string) (string, error) {
	var charData string
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		switch token := token.(type) {
		case xml.CharData:
			charData = string(token)
		case xml.EndElement:
			if token.Name.Local == endElementName {
				return charData, nil
			}
		}
	}
	return "", &nodeUnclosedError{nodeName: endElementName}
}

// parseString reads character data from an XML decoder for the specified key
// and retrieves the corresponding value from the provided EnvValueProvider.
func parseString(key string, decoder *xml.Decoder, provider EnvValueProvider) (string, error) {
	value, err := parseCharData(decoder, key)
	if err != nil {
		return "", err
	}
	return provider.Get(value)
}

// parseInt reads character data from an XML decoder for the specified key,
// retrieves the corresponding value from the provided EnvValueProvider,
// and converts it to an integer.
func parseInt(key string, decoder *xml.Decoder, provider EnvValueProvider) (int, error) {
	value, err := parseCharData(decoder, key)
	if err != nil {
		return 0, err
	}
	str, err := provider.Get(value)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str)
}
