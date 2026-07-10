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

// Document is the format-independent representation of a Juice configuration.
type Document struct {
	Settings         map[string]string
	Environments     Environments
	MapperAttributes map[string]string
	MapperSources    []MapperSource
	MapperEntries    []MapperEntry
	Mappers          []Mapper
}

// Environments contains the configured database environments.
type Environments struct {
	Default string
	Items   []Environment
	Present bool
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

// MapperSource identifies mapper documents referenced by the configuration.
type MapperSource struct {
	Resource string
	URL      string
	Pattern  string
}

// MapperEntry preserves the declaration order of mapper sources and inline mappers.
type MapperEntry struct {
	Source *MapperSource
	Mapper *Mapper
}

// Mapper is a parsed mapper document.
type Mapper struct {
	Namespace  string
	Attributes map[string]string
	Statements []Statement
	Fragments  []Fragment
}

// Fragment is a reusable SQL node group declared by a sql element.
type Fragment struct {
	ID    string
	Nodes []Node
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
	Nodes      []Node
}
