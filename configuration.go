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
	"errors"
	"fmt"
	"io/fs"
	"iter"
	"os"
	unixpath "path"
	"path/filepath"
	"reflect"

	"github.com/go-juicedev/juice/internal/rootfs"
	configparser "github.com/go-juicedev/juice/parser"
	xmlparser "github.com/go-juicedev/juice/parser/xml"
)

var (
	errConfigurationPathRequired              = errors.New("configuration path is required")
	errConfigurationRequired                  = errors.New("configuration is required")
	errConfigurationEnvironmentsRequired      = errors.New("environments section is required")
	errConfigurationDefaultEnvironmentMissing = errors.New("default environment is not specified")
	errConfigurationDefaultEnvironmentUnknown = errors.New("default environment not found")
	errConfigurationBackendRequired           = errors.New("script backend is required")
	errMapperRootElementNotFound              = xmlparser.ErrMapperRootElementNotFound
)

// RuntimeConfiguration provides the settings and sources needed to start
// an Engine. It contains configuration data only and owns no live resources.
type RuntimeConfiguration interface {
	SourceProvider

	// Settings returns the settings.
	Settings() SettingProvider
}

// BackendProvider provides the script backend used to compile raw statements.
type BackendProvider interface {
	// Backend returns the syntax backend used by the configuration.
	Backend() configparser.Backend
}

// Configuration is the compiled configuration consumed when an Engine starts.
// Its smaller component interfaces let runtime consumers depend only on the
// configuration facet they actually use.
type Configuration interface {
	StatementCatalog
	RuntimeConfiguration
	BackendProvider
}

// CompiledConfig is an immutable configuration artifact produced by Compile.
// It contains no open database connections or other live runtime resources.
type CompiledConfig struct {
	// backend provides the syntax-specific script behavior.
	backend configparser.Backend

	// catalog contains compiled statements indexed by canonical ID.
	catalog *statementCatalog

	// runtime contains compiled settings and database source definitions.
	runtime *RuntimeConfig
}

func (c *CompiledConfig) validate(ignoreEnv bool) error {
	if c.backend == nil {
		return errConfigurationBackendRequired
	}
	if !ignoreEnv {
		if c.runtime == nil || len(c.runtime.sources) == 0 {
			return errConfigurationEnvironmentsRequired
		}
		defaultSource := c.runtime.DefaultSource()
		if defaultSource == "" {
			return errConfigurationDefaultEnvironmentMissing
		}
		if _, exists := c.runtime.Source(defaultSource); !exists {
			return fmt.Errorf("%w: %s", errConfigurationDefaultEnvironmentUnknown, defaultSource)
		}
	}

	return nil
}

// Settings returns the settings.
func (c CompiledConfig) Settings() SettingProvider {
	return c.runtime.Settings()
}

// DefaultSource returns the configured default database source.
func (c CompiledConfig) DefaultSource() string {
	return c.runtime.DefaultSource()
}

// Source returns a compiled database source by name.
func (c CompiledConfig) Source(name string) (Source, bool) {
	return c.runtime.Source(name)
}

// Sources iterates over compiled database sources.
func (c CompiledConfig) Sources() iter.Seq2[string, Source] {
	return c.runtime.Sources()
}

// Backend returns the syntax backend associated with the configuration.
func (c CompiledConfig) Backend() configparser.Backend {
	return c.backend
}

// resolveStatementID derives a canonical statement ID from a lookup value.
func resolveStatementID(v any) (StatementID, error) {
	if v == nil {
		return "", errors.New("nil statement query")
	}

	var id StatementID
	// If v implements StatementID(), use it directly.
	// If v is a string, treat it as the statement ID.
	// Otherwise, derive the statement ID via reflection.
	switch t := v.(type) {
	case interface{ StatementID() string }:
		id = StatementID(t.StatementID())
	case StatementID:
		id = t
	case string:
		id = StatementID(t)
	default:
		// Derive the statement ID from the reflected value.
		rv := reflect.Indirect(reflect.ValueOf(v))
		switch rv.Kind() {
		case reflect.Func:
			id = StatementID(cachedRuntimeFuncName(rv.Pointer()))
		case reflect.Struct:
			id = StatementID(rv.Type().PkgPath() + "." + rv.Type().Name())
		default:
			return "", fmt.Errorf("cannot extract statement ID from value of type %T: must be string, StatementID() string interface, or struct/func", v)
		}
	}

	if len(id) == 0 {
		return "", fmt.Errorf("cannot extract statement ID from value of type %T", v)
	}
	return id, nil
}

// Statement returns a compiled statement by canonical ID.
func (c CompiledConfig) Statement(id StatementID) (Statement, error) {
	return c.catalog.Statement(id)
}

// GetStatement resolves legacy string, function, and struct lookup values.
// New code should prefer Statement with an explicit StatementID.
func (c CompiledConfig) GetStatement(v any) (Statement, error) {
	id, err := resolveStatementID(v)
	if err != nil {
		return nil, err
	}
	return c.Statement(id)
}

// NewXMLConfiguration parses and compiles an XML configuration file.
func NewXMLConfiguration(filename string) (*CompiledConfig, error) {
	return newLocalXMLConfiguration(filename, false)
}

// Used by go:linkname.
func newLocalXMLConfiguration(filename string, ignoreEnv bool) (*CompiledConfig, error) {
	if filename == "" {
		return nil, errConfigurationPathRequired
	}
	dirname := filepath.Dir(filename)
	filename = filepath.Base(filename)
	root, err := os.OpenRoot(dirname)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	return compileXMLConfiguration(root.FS(), filename, ignoreEnv)
}

// NewXMLConfigurationWithFS parses and compiles an XML configuration from fs.
// The filepath parameter must be a Unix-style path (using forward slashes '/'),
// because it is processed with path.Dir and path.Base.
func NewXMLConfigurationWithFS(fs fs.FS, filepath string) (*CompiledConfig, error) {
	if filepath == "" {
		return nil, errConfigurationPathRequired
	}
	root := unixpath.Dir(filepath)
	filename := unixpath.Base(filepath)
	return compileXMLConfiguration(rootfs.New(fs, root), filename, false)
}

// compileXMLConfiguration parses and compiles an XML file.
// When ignoreEnv is true, the <environments> section is skipped.
// For internal use only.
func compileXMLConfiguration(fs fs.FS, filepath string, ignoreEnv bool) (*CompiledConfig, error) {
	parser := &xmlparser.Parser{
		FS:                fs,
		IgnoreEnvironment: ignoreEnv,
	}
	document, err := parser.ParseFile(filepath)
	if err != nil {
		if errors.Is(err, xmlparser.ErrMapperRootElementNotFound) {
			return nil, errors.Join(errMapperRootElementNotFound, err)
		}
		return nil, err
	}
	return Compile(document, CompileOptions{
		Backend:           xmlparser.Backend{},
		IgnoreEnvironment: ignoreEnv,
	})
}

var _ Configuration = (*CompiledConfig)(nil)
