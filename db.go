/*
Copyright 2025 eatmoreapple

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

// Package juice provides a robust and thread-safe database connection management system.
// It supports multiple database drivers and connection pooling with configurable parameters.
package juice

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-juicedev/juice/driver"
)

// Source encapsulates all configuration parameters needed for establishing
// and maintaining a database connection. It includes connection pool settings
// and lifecycle management parameters.
type Source struct {
	Driver          string
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// conn represents an active database connection along with its associated driver.
// It uses sync.Once to ensure thread-safe initialization.
type conn struct {
	db   *sql.DB
	drv  driver.Driver
	err  error
	once sync.Once
}

// DBManager implements a thread-safe connection manager for multiple database instances.
// It maintains a pool of connections and their corresponding configurations.
type DBManager struct {
	conns   sync.Map          // active connections
	sources map[string]Source // connection sources
	mu      sync.RWMutex      // protects sources map
	closed  atomic.Bool       // manager state
	names   []string          // sorted list of registered sources
}

var (
	// ErrDBManagerClosed is returned when attempting to use a closed manager
	ErrDBManagerClosed = errors.New("juice: manager is closed")

	// ErrSourceExists is returned when attempting to add a duplicate source
	ErrSourceExists = errors.New("juice: source already exists")

	// ErrSourceNotFound is returned when attempting to access a non-existent source
	ErrSourceNotFound = errors.New("juice: source not found")
)

// Get retrieves an existing database connection or creates a new one if it doesn't exist.
// It returns the database connection, its driver, and any error that occurred.
// This method is thread-safe and ensures only one connection is created per source.
func (m *DBManager) Get(name string) (*sql.DB, driver.Driver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed.Load() {
		return nil, nil, ErrDBManagerClosed
	}

	source, exists := m.sources[name]
	if !exists {
		return nil, nil, fmt.Errorf("%w: %s", ErrSourceNotFound, name)
	}

	return m.connect(name, source)
}

// connect establishes a new database connection using the provided source configuration.
// It ensures thread-safe connection initialization using sync.Once and properly
// configures connection pool parameters.
func (m *DBManager) connect(name string, source Source) (db *sql.DB, drv driver.Driver, err error) {
	actual, _ := m.conns.LoadOrStore(name, &conn{})
	c := actual.(*conn)

	c.once.Do(func() {
		c.drv, c.err = driver.Get(source.Driver)
		if c.err != nil {
			c.err = fmt.Errorf("failed to get driver: %w", c.err)
			return
		}
		c.db, c.err = driver.Connect(
			source.Driver,
			source.DSN,
			driver.ConnectWithMaxOpenConnNum(source.MaxOpenConns),
			driver.ConnectWithMaxIdleConnNum(source.MaxIdleConns),
			driver.ConnectWithMaxConnLifetime(source.ConnMaxLifetime),
			driver.ConnectWithMaxIdleConnLifetime(source.ConnMaxIdleTime),
		)
		if c.err != nil {
			c.err = fmt.Errorf("failed to create connection: %w", c.err)
			return
		}
	})

	if c.err != nil {
		m.conns.Delete(name)
		return nil, nil, c.err
	}

	return c.db, c.drv, nil
}

// Add registers a new database source configuration with the manager.
// It returns an error if the source already exists or if the manager is closed.
// This method is thread-safe and prevents duplicate source registration.
func (m *DBManager) Add(name string, source Source) error {

	if m.closed.Load() {
		return ErrDBManagerClosed
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed.Load() {
		return ErrDBManagerClosed
	}

	if m.sources == nil {
		m.sources = make(map[string]Source)
	}

	if _, exists := m.sources[name]; exists {
		return fmt.Errorf("%w: %s", ErrSourceExists, name)
	}

	m.sources[name] = source
	m.names = append(m.names, name)
	return nil
}

func (m *DBManager) Registered() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Clone(m.names)
}

// Close gracefully shuts down all managed database connections.
// It ensures that all resources are properly released and prevents new connections
// from being established. This method is idempotent and thread-safe.
func (m *DBManager) Close() error {
	if m.closed.Load() {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed.Load() {
		return nil
	}

	m.closed.Store(true)

	var errs []error
	m.conns.Range(func(key, value interface{}) bool {
		c := value.(*conn)
		if c.db == nil {
			return true
		}
		if err := c.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %v: %w", key, err))
		}
		return true
	})

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// NewDBManager creates a new DBManager instance using the provided
// configuration. It initializes all configured database sources and validates their
// parameters before adding them to the manager.
func NewDBManager(cfg Configuration) (*DBManager, error) {
	if cfg == nil {
		return nil, errConfigurationRequired
	}

	envs := cfg.Environments()
	if envs == nil {
		return nil, errConfigurationEnvironmentsRequired
	}

	defaultEnv := envs.Attribute("default")
	if defaultEnv == "" {
		return nil, errConfigurationDefaultEnvironmentMissing
	}
	if _, err := envs.Use(defaultEnv); err != nil {
		return nil, fmt.Errorf("%w: %s", errConfigurationDefaultEnvironmentUnknown, defaultEnv)
	}

	m := &DBManager{
		sources: make(map[string]Source),
	}

	for name, env := range envs.Iter() {
		if err := m.Add(name, Source{
			Driver:          env.Driver,
			DSN:             env.DataSource,
			MaxOpenConns:    env.MaxOpenConnNum,
			MaxIdleConns:    env.MaxIdleConnNum,
			ConnMaxLifetime: time.Duration(env.MaxConnLifetime) * time.Second,
			ConnMaxIdleTime: time.Duration(env.MaxIdleConnLifetime) * time.Second,
		}); err != nil {
			return nil, fmt.Errorf("failed to add source %s: %w", name, err)
		}
	}

	return m, nil
}
