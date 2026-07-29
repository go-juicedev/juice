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

	"github.com/go-juicedev/juice/internal/container"
)

// StatementID is the fully qualified identifier of a mapped statement.
// Its canonical form is "[prefix.]namespace.statement".
type StatementID string

// String returns the canonical statement ID.
func (id StatementID) String() string { return string(id) }

// StatementCatalog provides immutable statement lookup by canonical ID.
type StatementCatalog interface {
	// Statement returns the statement with the given canonical ID.
	Statement(id StatementID) (Statement, error)
}

type statementCatalog struct {
	// Trie shares the dot-delimited namespace prefixes across statement IDs.
	statements *container.Trie[Statement]
}

func newStatementCatalog() *statementCatalog {
	return &statementCatalog{statements: container.NewTrie[Statement]()}
}

func (c *statementCatalog) add(statement Statement) error {
	id := statement.ID()
	if id == "" {
		return fmt.Errorf("%w: empty statement id", ErrNoStatementFound)
	}
	key := id.String()
	if _, exists := c.statements.Get(key); exists {
		return fmt.Errorf("duplicate statement id: %s", id)
	}
	c.statements.Insert(key, statement)
	return nil
}

func (c *statementCatalog) Statement(id StatementID) (Statement, error) {
	if c == nil || c.statements == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoStatementFound, id)
	}
	statement, exists := c.statements.Get(id.String())
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrNoStatementFound, id)
	}
	return statement, nil
}

var _ StatementCatalog = (*statementCatalog)(nil)
