package juice

import (
	"fmt"
	"strings"

	"github.com/go-juicedev/juice/internal/container"
	"github.com/go-juicedev/juice/node"
)

// Mapper defines a set of statements.
type Mapper struct {
	namespace  string
	mappers    *Mappers
	statements map[string]*xmlSQLStatement
	sqlNodes   map[string]*node.SQLNode
	attrs      map[string]string
}

// Namespace returns the namespace of the mapper.
func (m *Mapper) Namespace() string {
	return m.namespace
}

func (m *Mapper) setAttribute(key, value string) {
	if m.attrs == nil {
		m.attrs = make(map[string]string)
	}
	m.attrs[key] = value
}

func (m *Mapper) setSqlNode(n *node.SQLNode) error {
	if m.sqlNodes == nil {
		m.sqlNodes = make(map[string]*node.SQLNode)
	}
	if _, exists := m.sqlNodes[n.ID]; exists {
		return fmt.Errorf("sql node %s already exists", n.ID)
	}
	m.sqlNodes[n.ID] = n
	return nil
}

// Attribute returns the attribute value by key.
func (m *Mapper) Attribute(key string) string {
	return m.attrs[key]
}

func (m *Mapper) GetSQLNodeByID(id string) (node.Node, error) {
	// if the id is not cross-namespace
	isCrossNamespace := strings.Contains(id, ".")

	if !isCrossNamespace {
		n, exists := m.sqlNodes[id]
		if !exists {
			return nil, fmt.Errorf("SQL node %q not found in mapper %q", id, m.namespace)
		}
		return n, nil
	}

	return m.mappers.GetSQLNodeByID(id)
}

func (m *Mapper) GetStatementByID(id string) (Statement, bool) {
	statement, exists := m.statements[id]
	return statement, exists
}

// Mappers is a container for all mappers.
type Mappers struct {
	attrs map[string]string
	cfg   Configuration
	// mappers uses Trie instead of map because mapper namespaces often share common prefixes
	// (e.g., "com.example.user", "com.example.order"). Trie provides both memory efficiency
	// by storing shared prefixes only once and fast prefix-based lookups
	mappers *container.Trie[*Mapper]
}

func (m *Mappers) setMapper(key string, mapper *Mapper) error {
	if prefix := m.Prefix(); prefix != "" {
		key = fmt.Sprintf("%s.%s", prefix, key)
	}
	if m.mappers == nil {
		m.mappers = container.NewTrie[*Mapper]()
	}
	if _, exists := m.mappers.Get(key); exists {
		return fmt.Errorf("mapper %s already exists", key)
	}
	mapper.mappers = m
	m.mappers.Insert(key, mapper)
	return nil
}

func (m *Mappers) GetMapperByNamespace(namespace string) (*Mapper, bool) {
	if m == nil || m.mappers == nil {
		return nil, false
	}
	return m.mappers.Get(namespace)
}

func (m *Mappers) getMapperAndNodeID(id string) (mapper *Mapper, key string, err error) {
	lastDotIndex := strings.LastIndex(id, ".")
	if lastDotIndex <= 0 {
		return nil, "", fmt.Errorf("mapper id %q does not have a .id", id)
	}

	namespace, nodeID := id[:lastDotIndex], id[lastDotIndex+1:]

	mapper, exists := m.GetMapperByNamespace(namespace)
	if !exists {
		return nil, "", fmt.Errorf("mapper %s not found", namespace)
	}
	return mapper, nodeID, nil
}

// GetStatementByID returns a Statement by id.
// The id should be in the format of "namespace.statementName"
// For example: "main.UserMapper.SelectUser"
func (m *Mappers) GetStatementByID(id string) (Statement, error) {
	if m == nil {
		return nil, fmt.Errorf("%w: statement '%s' not found in mapper configuration", ErrNoStatementFound, id)
	}

	mapper, statementID, err := m.getMapperAndNodeID(id)
	if err != nil {
		return nil, err
	}

	statement, exists := mapper.GetStatementByID(statementID)
	if !exists {
		return nil, fmt.Errorf("statement '%s' not found in namespace '%s'", statementID, mapper.Namespace())
	}

	return statement, nil
}

func (m *Mappers) GetSQLNodeByID(id string) (node.Node, error) {
	mapper, sqlNodeID, err := m.getMapperAndNodeID(id)
	if err != nil {
		return nil, err
	}
	return mapper.GetSQLNodeByID(sqlNodeID)
}

// Configuration represents a configuration of juice.
func (m *Mappers) Configuration() Configuration {
	return m.cfg
}

// setAttribute sets an attribute.
// same as setAttribute, but it is used for Mappers.
func (m *Mappers) setAttribute(key, value string) {
	if m.attrs == nil {
		m.attrs = make(map[string]string)
	}
	m.attrs[key] = value
}

// Attribute returns an attribute from the Mappers attributes.
func (m *Mappers) Attribute(key string) string {
	return m.attrs[key]
}

// Prefix returns the prefix of the Mappers.
func (m *Mappers) Prefix() string {
	return m.Attribute("prefix")
}
