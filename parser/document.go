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

package parser

import "github.com/go-juicedev/juice/node"

// Document is the format-independent representation of a Juice configuration.
type Document struct {
	Settings     map[string]string
	Environments Environments
	MapperPrefix string
	Mappers      []Mapper
}

// Environments contains the configured database environments.
type Environments struct {
	Default string
	Items   []Environment
}

// Environment describes one database environment before semantic compilation.
type Environment struct {
	ID                  string
	Driver              string
	DataSource          string
	MaxIdleConns        string
	MaxOpenConns        string
	ConnMaxLifetime     string
	ConnMaxIdleLifetime string
	Attributes          map[string]string
}

// Mapper is a parsed mapper document.
type Mapper struct {
	Namespace  string
	Statements []Statement
}

// Action identifies the operation represented by a statement.
type Action string

const (
	Select Action = "select"
	Insert Action = "insert"
	Update Action = "update"
	Delete Action = "delete"
)

// Statement is a mapped SQL statement before it is compiled for execution.
type Statement struct {
	ID         string
	Action     Action
	Attributes map[string]string
	Node       node.Node
}
