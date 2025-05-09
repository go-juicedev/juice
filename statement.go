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
	"hash/fnv"
	"strconv"

	"github.com/go-juicedev/juice/driver"
)

type Statement interface {
	ID() string
	Name() string
	Attribute(key string) string
	Action() action
	Configuration() IConfiguration
	ResultMap() (ResultMap, error)
	Build(translator driver.Translator, param Param) (query string, args []any, err error)
}

// xmlSQLStatement defines a sql xmlSQLStatement.
type xmlSQLStatement struct {
	mapper *Mapper
	action action
	Nodes  NodeGroup
	attrs  map[string]string
	name   string
	id     string
}

// Attribute returns the value of the attribute with the given key.
func (s *xmlSQLStatement) Attribute(key string) string {
	value := s.attrs[key]
	if value == "" {
		value = s.mapper.Attribute(key)
	}
	return value
}

// setAttribute sets the attribute with the given key and value.
func (s *xmlSQLStatement) setAttribute(key, value string) {
	if s.attrs == nil {
		s.attrs = make(map[string]string)
	}
	s.attrs[key] = value
}

// ID returns the unique key of the namespace.
func (s *xmlSQLStatement) ID() string {
	return s.id
}

func (s *xmlSQLStatement) lazyName() string {
	var builder = getStringBuilder()
	defer putStringBuilder(builder)
	if prefix := s.mapper.mappers.Prefix(); prefix != "" {
		builder.WriteString(prefix)
		builder.WriteString(".")
	}
	builder.WriteString(s.mapper.namespace)
	builder.WriteString(".")
	builder.WriteString(s.id)
	return builder.String()
}

// Name is a unique key of the whole xmlSQLStatement.
func (s *xmlSQLStatement) Name() string {
	if s.name == "" {
		s.name = s.lazyName()
	}
	return s.name
}

// Action returns the action of the xmlSQLStatement.
func (s *xmlSQLStatement) Action() action {
	return s.action
}

// Configuration returns the configuration of the xmlSQLStatement.
func (s *xmlSQLStatement) Configuration() IConfiguration {
	return s.mapper.mappers.Configuration()
}

// ResultMap returns the ResultMap of the xmlSQLStatement.
func (s *xmlSQLStatement) ResultMap() (ResultMap, error) {
	// TODO: implement the ResultMap method.
	// why is this not implemented?
	// result map implementation is too complex, and it's not a common feature.
	return nil, ErrResultMapNotSet
}

// Build builds the xmlSQLStatement with the given parameter.
func (s *xmlSQLStatement) Build(translator driver.Translator, param Param) (query string, args []any, err error) {
	value := newGenericParam(param, s.Attribute("paramName"))
	query, args, err = s.Nodes.Accept(translator, value)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", nil, ErrEmptyQuery
	}
	return query, args, nil
}

// rawSQLStatement represents a raw SQL query with its parameters and action type.
// It implements the Statement interface and provides methods for query execution.
type rawSQLStatement struct {
	query  string
	cfg    IConfiguration
	action action
}

// hash generates a unique 64-bit FNV-1a hash of the SQL query.
// This hash is used for both ID and Name generation.
func (s rawSQLStatement) hash() uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s.query))
	return h.Sum64()
}

// ID returns a unique identifier for the statement.
// Format: "id:" + hexadecimal hash of the query
func (s rawSQLStatement) ID() string {
	return "id:" + strconv.FormatUint(s.hash(), 16)
}

// Name returns a hexadecimal representation of the query hash.
// Used for identifying the statement in logs and debugging.
func (s rawSQLStatement) Name() string {
	return strconv.FormatUint(s.hash(), 16)
}

// Attribute returns an empty string as rawSQLStatement does not support attributes.
func (s rawSQLStatement) Attribute(_ string) string {
	return ""
}

// Action returns the action of the rawSQLStatement.
func (s rawSQLStatement) Action() action {
	return s.action
}

// Configuration returns the configuration of the rawSQLStatement.
func (s rawSQLStatement) Configuration() IConfiguration {
	return s.cfg
}

// ResultMap returns the ResultMap of the rawSQLStatement.
func (s rawSQLStatement) ResultMap() (ResultMap, error) {
	// TODO: implement the ResultMap method.
	return nil, ErrResultMapNotSet
}

// Build builds the rawSQLStatement with the given parameter.
func (s rawSQLStatement) Build(translator driver.Translator, param Param) (query string, args []any, err error) {
	value := newGenericParam(param, "")
	query, args, err = NewTextNode(s.query).Accept(translator, value)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", nil, ErrEmptyQuery
	}
	return query, args, nil
}

// NewRawSQLStatement creates a new raw SQL statement with the given query, configuration, and action.
func NewRawSQLStatement(query string, cfg IConfiguration, action action) Statement {
	return &rawSQLStatement{
		query:  query,
		cfg:    cfg,
		action: action,
	}
}
