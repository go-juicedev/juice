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
	"os"
	unixpath "path"
	"path/filepath"
	"reflect"
)

var errConfigurationPathRequired = errors.New("configuration path is required")

// Configuration provides access to environments, settings, and mapped statements.
type Configuration interface {
	// Environments returns the environments.
	Environments() EnvironmentProvider

	// Settings returns the settings.
	Settings() SettingProvider

	// GetStatement returns the statement associated with the given value.
	GetStatement(v any) (Statement, error)
}

// xmlConfiguration is the XML-backed implementation of Configuration.
type xmlConfiguration struct {
	// environments is a map of environments.
	environments *environments

	// mappers is a map of mappers.
	mappers *Mappers

	// settings is a map of settings.
	settings keyValueSettingProvider
}

// Environments returns the environments.
func (c xmlConfiguration) Environments() EnvironmentProvider {
	return c.environments
}

// Settings returns the settings.
func (c xmlConfiguration) Settings() SettingProvider {
	return &c.settings
}

// GetStatement returns the statement associated with the given value.
func (c xmlConfiguration) GetStatement(v any) (Statement, error) {
	if v == nil {
		return nil, errors.New("nil statement query")
	}

	var id string
	// If v implements StatementID(), use it directly.
	// If v is a string, treat it as the statement ID.
	// Otherwise, derive the statement ID via reflection.
	switch t := v.(type) {
	case interface{ StatementID() string }:
		id = t.StatementID()
	case string:
		id = t
	default:
		// Derive the statement ID from the reflected value.
		rv := reflect.Indirect(reflect.ValueOf(v))
		switch rv.Kind() {
		case reflect.Func:
			id = cachedRuntimeFuncName(rv.Pointer())
		case reflect.Struct:
			id = rv.Type().PkgPath() + "." + rv.Type().Name()
		default:
			return nil, fmt.Errorf("cannot extract statement ID from value of type %T: must be string, StatementID() string interface, or struct/func", v)
		}
	}

	if len(id) == 0 {
		return nil, fmt.Errorf("cannot extract statement ID from value of type %T", v)
	}

	return c.mappers.GetStatementByID(id)
}

func NewXMLConfiguration(filename string) (Configuration, error) {
	return newLocalXMLConfiguration(filename, false)
}

// Used by go:linkname.
func newLocalXMLConfiguration(filename string, ignoreEnv bool) (Configuration, error) {
	if filename == "" {
		return nil, errConfigurationPathRequired
	}
	dirname := filepath.Dir(filename)
	filename = filepath.Base(filename)
	root, err := os.OpenRoot(dirname)
	if err != nil {
		return nil, err
	}
	return newXMLConfigurationParser(root.FS(), filename, ignoreEnv)
}

// NewXMLConfigurationWithFS creates a new XML configuration parser with a given fs.FS and filename.
// The filepath parameter must be a Unix-style path (using forward slashes '/'),
// as it will be processed by path.Dir and path.Base.
func NewXMLConfigurationWithFS(fs fs.FS, filepath string) (Configuration, error) {
	if filepath == "" {
		return nil, errConfigurationPathRequired
	}
	basedir := unixpath.Dir(filepath)
	filename := unixpath.Base(filepath)
	return newXMLConfigurationParser(newFsRoot(fs, basedir), filename, false)
}

// newXMLConfigurationParser creates a configuration parser for an XML file.
// When ignoreEnv is true, the <environments> section is skipped.
// For internal use only.
func newXMLConfigurationParser(fs fs.FS, filepath string, ignoreEnv bool) (Configuration, error) {
	file, err := fs.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	parser := &XMLParser{FS: fs, ignoreEnv: ignoreEnv}
	parser.AddXMLElementParser(
		&XMLEnvironmentsElementParser{},
		&XMLMappersElementParser{},
		&XMLSettingsElementParser{},
	)
	return parser.Parse(file)
}
