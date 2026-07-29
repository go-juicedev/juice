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
	"context"
	"database/sql"
	"io/fs"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/parser"
)

// Engine is the implementation of Manager interface and the core of juice.
type Engine struct {
	catalog  StatementCatalog
	sources  SourceProvider
	settings SettingProvider
	backend  parser.Backend

	// driver translates statements and opens database connections.
	driver driver.Driver

	// db is the database connection
	db *sql.DB

	// using is the active environment id.
	using string

	manager *DBManager

	// middlewares intercept statement execution for logging, tracing, routing, and similar concerns.
	middlewares MiddlewareGroup
}

// executor creates an SQLRowsExecutor for the mapped statement.
func (e *Engine) executor(v any) (SQLRowsExecutor, error) {
	statement, err := e.resolveStatement(v)
	if err != nil {
		return nil, err
	}
	statementHandler := newBatchStatementHandler(e, e.DB())
	return NewSQLRowsExecutor(statement, statementHandler, e.Driver()), nil
}

func (e *Engine) resolveStatement(v any) (Statement, error) {
	id, err := resolveStatementID(v)
	if err != nil {
		return nil, err
	}
	return e.catalog.Statement(id)
}

// Object implements the Manager interface
func (e *Engine) Object(v any) SQLRowsExecutor {
	exe, err := e.executor(v)
	if err != nil {
		return inValidExecutor(err)
	}
	return exe
}

// Tx returns a TxManager
func (e *Engine) Tx() *BasicTxManager {
	return e.ContextTx(context.Background(), nil)
}

// ContextTx returns a TxManager with the given context
func (e *Engine) ContextTx(ctx context.Context, opt *sql.TxOptions) *BasicTxManager {
	return &BasicTxManager{
		basicTxManager: &basicTxManager{
			engine: e,
			ctx:    ctx,
		},
		txOptions: opt,
	}
}

// Settings returns the settings used by the engine.
func (e *Engine) Settings() SettingProvider {
	return e.settings
}

// Backend returns the script backend used to compile raw statements.
func (e *Engine) Backend() parser.Backend {
	return e.backend
}

// Use adds a middleware to the engine
func (e *Engine) Use(middleware Middleware) {
	e.middlewares = append(e.middlewares, middleware)
}

func (e *Engine) clone() *Engine {
	return &Engine{
		catalog:     e.catalog,
		sources:     e.sources,
		settings:    e.settings,
		backend:     e.backend,
		manager:     e.manager,
		middlewares: e.middlewares,
	}
}

// With creates a new Engine instance with the specified environment name.
// If the requested environment name matches the current one, it returns the same engine.
// Otherwise, it creates a cloned engine with the new database connection and driver.
// Returns an error if the specified environment is not found or connection fails.
func (e *Engine) With(name string) (*Engine, error) {
	if e.using == name {
		return e, nil
	}
	db, drv, err := e.manager.Get(name)
	if err != nil {
		return nil, err
	}
	engine := e.clone()
	engine.db, engine.driver = db, drv
	engine.using = name
	return engine, nil
}

// EnvID returns the identifier of the currently active database environment.
func (e *Engine) EnvID() string {
	return e.using
}

// DB returns the database connection of the engine
func (e *Engine) DB() *sql.DB {
	return e.db
}

// Driver returns the driver of the engine
func (e *Engine) Driver() driver.Driver {
	return e.driver
}

// Close gracefully shuts down all managed database connections
// all cloned engines share the same DBManager
func (e *Engine) Close() error {
	return e.manager.Close()
}

// init initializes the engine
func (e *Engine) init() (err error) {
	e.manager, err = NewDBManager(e.sources)
	if err != nil {
		return
	}
	e.using = e.sources.DefaultSource()
	e.db, e.driver, err = e.manager.Get(e.using)
	return err
}

func (e *Engine) Raw(query string) Runner {
	return NewRunner(query, e, e.DB())
}

// New is the alias of NewEngine
func New(configuration Configuration) (*Engine, error) {
	if configuration == nil {
		return nil, errConfigurationRequired
	}
	backend := configuration.Backend()
	if backend == nil {
		return nil, errConfigurationBackendRequired
	}
	settings := configuration.Settings()
	if settings == nil {
		settings = keyValueSettingProvider{}
	}
	engine := &Engine{
		catalog:  configuration,
		sources:  configuration,
		settings: settings,
		backend:  backend,
	}
	if err := engine.init(); err != nil {
		return nil, err
	}
	// add the default middlewares
	engine.Use(&useGeneratedKeysMiddleware{})
	return engine, nil
}

// NewFromFile creates a new Engine from a configuration file path.
// It automatically creates the configuration from the file and initializes the engine.
func NewFromFile(filename string) (*Engine, error) {
	config, err := NewXMLConfiguration(filename)
	if err != nil {
		return nil, err
	}
	return New(config)
}

// NewFromFS creates a new Engine from a filesystem and configuration file path.
// It automatically creates the configuration and initializes the engine.
func NewFromFS(fs fs.FS, filepath string) (*Engine, error) {
	config, err := NewXMLConfigurationWithFS(fs, filepath)
	if err != nil {
		return nil, err
	}
	return New(config)
}

// Default creates a new Engine with the default middlewares
// It adds an interceptor to log the statements
func Default(configuration Configuration) (*Engine, error) {
	engine, err := New(configuration)
	if err != nil {
		return nil, err
	}
	engine.Use(&TimeoutMiddleware{})
	engine.Use(&DebugMiddleware{})
	return engine, nil
}

// DefaultFromFile creates a new Engine from a configuration file path with default middlewares.
// It automatically creates the configuration from the file and initializes the engine.
func DefaultFromFile(filename string) (*Engine, error) {
	config, err := NewXMLConfiguration(filename)
	if err != nil {
		return nil, err
	}
	return Default(config)
}

// DefaultFromFS creates a new Engine from a filesystem and configuration file path with default middlewares.
// It automatically creates the configuration and initializes the engine.
func DefaultFromFS(fs fs.FS, filepath string) (*Engine, error) {
	config, err := NewXMLConfigurationWithFS(fs, filepath)
	if err != nil {
		return nil, err
	}
	return Default(config)
}
