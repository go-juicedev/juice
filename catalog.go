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
)

// StatementID is the fully qualified identifier of a mapped statement.
// Its canonical form is "namespace.statement".
type StatementID string

// String returns the canonical statement ID.
func (id StatementID) String() string { return string(id) }

func newStatementID(namespace, statement string) StatementID {
	return StatementID(namespace + "." + statement)
}

// StatementCatalog provides immutable statement lookup by canonical ID.
type StatementCatalog interface {
	// Statement returns the statement with the given canonical ID.
	Statement(id StatementID) (Statement, error)
}

type statementCatalog struct {
	statements map[StatementID]Statement
}

func newStatementCatalog() *statementCatalog {
	return &statementCatalog{}
}

func (c *statementCatalog) add(statement Statement) error {
	id := statement.ID()
	if id == "" {
		return fmt.Errorf("%w: empty statement id", ErrNoStatementFound)
	}
	if c.statements == nil {
		c.statements = make(map[StatementID]Statement)
	}
	if _, exists := c.statements[id]; exists {
		return fmt.Errorf("duplicate statement id: %s", id)
	}
	c.statements[id] = statement
	return nil
}

func (c *statementCatalog) Statement(id StatementID) (Statement, error) {
	statement, exists := c.statements[id]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrNoStatementFound, id)
	}
	return statement, nil
}

var _ StatementCatalog = (*statementCatalog)(nil)
