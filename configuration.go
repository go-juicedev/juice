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
	`errors`
	`fmt`
	"io/fs"
	"os"
	unixpath "path"
	"path/filepath"
	`reflect`
)

// Configuration is the interface of configuration.
type Configuration interface {
	// Environments returns the environments.
	Environments() EnvironmentProvider

	// Settings returns the settings.
	Settings() SettingProvider

	// GetStatement returns the xmlSQLStatement of the given value.
	GetStatement(v any) (Statement, error)
}

// xmlConfiguration is a configuration of juice.
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

// GetStatement returns the xmlSQLStatement of the given value.
func (c xmlConfiguration) GetStatement(v any) (Statement, error) {
	if v == nil {
		return nil, errors.New("nil statement query")
	}

	var id string
	// if the interface is StatementID, use the StatementID() method to get the id
	// or if the interface is a string type, use the string as the id
	// otherwise, use the reflection to get the id
	switch t := v.(type) {
	case interface{ StatementID() string }:
		id = t.StatementID()
	case string:
		id = t
	default:
		// else try to one the id from the interface
		rv := reflect.Indirect(reflect.ValueOf(v))
		switch rv.Kind() {
		case reflect.Func:
			id = cachedRuntimeFuncName(rv.Pointer())
		case reflect.Struct:
			id = rv.Type().PkgPath() + "." + rv.Type().Name()
		default:
			return nil, errors.New("invalid type of xmlSQLStatement id")
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

// for go linkname
func newLocalXMLConfiguration(filename string, ignoreEnv bool) (Configuration, error) {
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
	basedir := unixpath.Dir(filepath)
	filename := unixpath.Base(filepath)
	return newXMLConfigurationParser(newFsRoot(fs, basedir), filename, false)
}

// newXMLConfigurationParser creates a new xmlConfiguration from an XML file which ignores environment parsing.
// for internal use only.
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
