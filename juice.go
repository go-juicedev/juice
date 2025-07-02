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

	"github.com/go-juicedev/juice/driver"
)

// Engine is the implementation of Manager interface and the core of juice.
type Engine struct {
	// configuration is the configuration of the engine
	// It is used to initialize the engine and to one the mapper statements
	configuration IConfiguration

	// driver is the driver used by the engine
	// It is used to initialize the database connection and translate the mapper statements
	driver driver.Driver

	// db is the database connection
	db *sql.DB

	// current using of environment id
	using string

	manager *DBManager

	// middlewares is the middlewares of the engine
	// It is used to intercept the execution of the statements
	// like logging, tracing, etc.
	middlewares MiddlewareGroup
}

// sqlRowsExecutor represents a mapper sqlRowsExecutor with the given parameters
func (e *Engine) executor(v any) (SQLRowsExecutor, error) {
	statement, err := e.GetConfiguration().GetStatement(v)
	if err != nil {
		return nil, err
	}
	statementHandler := NewBatchStatementHandler(e.Driver(), e.DB(), e.middlewares...)
	return NewSQLRowsExecutor(statement, statementHandler, e.Driver()), nil
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

// GetConfiguration returns the configuration of the engine
func (e *Engine) GetConfiguration() IConfiguration {
	return e.configuration
}

// Use adds a middleware to the engine
func (e *Engine) Use(middleware Middleware) {
	e.middlewares = append(e.middlewares, middleware)
}

func (e *Engine) clone() *Engine {
	return &Engine{
		configuration: e.configuration,
		manager:       e.manager,
		middlewares:   e.middlewares,
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
	e.manager, err = NewDBManager(e.configuration)
	if err != nil {
		return
	}
	e.using = e.configuration.Environments().Attribute("default")
	e.db, e.driver, err = e.manager.Get(e.using)
	return err
}

func (e *Engine) Raw(query string) Runner {
	return NewRunner(query, e, e.DB())
}

// New is the alias of NewEngine
func New(configuration IConfiguration) (*Engine, error) {
	engine := &Engine{
		configuration: configuration,
	}
	if err := engine.init(); err != nil {
		return nil, err
	}
	// add the default middlewares
	engine.Use(&useGeneratedKeysMiddleware{})
	return engine, nil
}

// Default creates a new Engine with the default middlewares
// It adds an interceptor to log the statements
func Default(configuration IConfiguration) (*Engine, error) {
	engine, err := New(configuration)
	if err != nil {
		return nil, err
	}
	engine.Use(&TimeoutMiddleware{})
	engine.Use(&DebugMiddleware{})
	return engine, nil
}
