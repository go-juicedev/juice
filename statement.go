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
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/node"
	"github.com/go-juicedev/juice/parser"
	"github.com/go-juicedev/juice/sql"
)

type StatementMetadata interface {
	ID() StatementID
	Attribute(key string) string
}

type StatementBuilder interface {
	Build(translator driver.Translator, parameter eval.Parameter) (query string, args []any, err error)
}

type Statement interface {
	Action() sql.Action
	ResultMap() (sql.ResultMap, error)
	StatementMetadata
	StatementBuilder
}

// mappedStatement represents a SQL statement produced from mapper configuration.
type mappedStatement struct {
	action sql.Action
	Nodes  node.Group
	attrs  map[string]string
	id     StatementID
}

// Attribute returns the value of the attribute with the given key.
func (s *mappedStatement) Attribute(key string) string {
	return s.attrs[key]
}

// setAttribute sets the attribute with the given key and value.
func (s *mappedStatement) setAttribute(key, value string) {
	if s.attrs == nil {
		s.attrs = make(map[string]string)
	}
	s.attrs[key] = value
}

// ID returns the fully qualified statement identifier.
func (s *mappedStatement) ID() StatementID {
	return s.id
}

// Action returns the SQL action for the statement.
func (s *mappedStatement) Action() sql.Action {
	return s.action
}

// ResultMap returns the result mapping strategy for the statement.
func (s *mappedStatement) ResultMap() (sql.ResultMap, error) {
	// Design Decision: ResultMap is intentionally not implemented for mapped statements.
	// Rationale:
	//   1. Complexity: Full ResultMap implementation requires complex nested object mapping,
	//      association handling, and discriminator logic similar to MyBatis.
	//   2. Alternative: Users can achieve the same result using struct tags (column:"name")
	//      which is more idiomatic in Go and provides compile-time type safety.
	//   3. Usage: This feature is rarely needed in practice. Most use cases are covered by
	//      simple struct field mapping via tags.
	// If you need custom result mapping, consider implementing the sql.RowScanner interface
	// on your struct type for full control over the scanning process.
	return nil, sql.ErrResultMapNotSet
}

// Build renders the mapped statement with the provided parameters.
func (s *mappedStatement) Build(translator driver.Translator, parameter eval.Parameter) (query string, args []any, err error) {
	query, args, err = s.Nodes.Accept(translator, parameter)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", nil, fmt.Errorf("statement %q generated empty query after parameter processing: %w", s.ID(), ErrEmptyQuery)
	}
	return query, args, nil
}

// RawSQLStatement represents a raw SQL query with its parameters and action type.
// It implements the Statement interface and provides methods for query execution.
type RawSQLStatement struct {
	query  string
	script node.Node
	action sql.Action
	attrs  map[string]string
}

// hash generates a unique 64-bit FNV-1a hash of the SQL query.
// This hash is used for ID generation.
func (s RawSQLStatement) hash() uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s.query))
	return h.Sum64()
}

// ID returns a unique identifier for the statement.
// Format: "id:" + hexadecimal hash of the query
func (s RawSQLStatement) ID() StatementID {
	return StatementID("id:" + strconv.FormatUint(s.hash(), 16))
}

// Attribute returns the statement attribute value for key.
func (s RawSQLStatement) Attribute(key string) string {
	if s.attrs == nil {
		return ""
	}
	return s.attrs[key]
}

// Action returns the action of the RawSQLStatement.
func (s RawSQLStatement) Action() sql.Action {
	return s.action
}

// ResultMap returns the ResultMap of the RawSQLStatement.
func (s RawSQLStatement) ResultMap() (sql.ResultMap, error) {
	// Design Decision: ResultMap is not supported for raw SQL statements.
	// Use struct tags or implement sql.RowScanner for custom result mapping.
	return nil, sql.ErrResultMapNotSet
}

// Build renders the raw SQL statement with the provided parameters.
func (s RawSQLStatement) Build(translator driver.Translator, parameter eval.Parameter) (query string, args []any, err error) {
	query, args, err = s.script.Accept(translator, parameter)
	if err != nil {
		return "", nil, err
	}
	if len(query) == 0 {
		return "", nil, fmt.Errorf("raw SQL statement %q generated empty query after parameter processing: %w", s.ID(), ErrEmptyQuery)
	}
	return query, args, nil
}

// WithAttribute adds or updates a key-value pair to the statement's attribute map.
func (s *RawSQLStatement) WithAttribute(key, value string) *RawSQLStatement {
	if s.attrs == nil {
		s.attrs = make(map[string]string)
	}
	s.attrs[key] = value
	return s
}

// NewRawSQLStatement creates a raw SQL statement using the given syntax backend.
func NewRawSQLStatement(backend parser.Backend, query string, action sql.Action) *RawSQLStatement {
	return &RawSQLStatement{
		query:  query,
		script: backend.Script(query),
		action: action,
	}
}
