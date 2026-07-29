package juice

import (
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"sync/atomic"
	"testing"

	jdriver "github.com/go-juicedev/juice/driver"
	configparser "github.com/go-juicedev/juice/parser"
	xmlparser "github.com/go-juicedev/juice/parser/xml"
)

type dbManagerDriver struct {
	name string
}

func (d dbManagerDriver) Translator() jdriver.Translator {
	return jdriver.TranslateFunc(func(_ string) string { return "?" })
}

func (d dbManagerDriver) Name() string {
	return d.name
}

func registerDBManagerTestDriver(t *testing.T) string {
	t.Helper()

	name := fmt.Sprintf("juice_db_manager_test_%d", atomic.AddUint64(&shSQLDriverSeq, 1))
	sql.Register(name, &shSQLDriver{state: &shSQLDriverState{}})
	jdriver.Register(name, dbManagerDriver{name: name})
	return name
}

func TestDBManagerRegisteredReturnsClone(t *testing.T) {
	manager := &DBManager{}

	if err := manager.Add("primary", Source{}); err != nil {
		t.Fatalf("Add(primary) error = %v", err)
	}
	if err := manager.Add("secondary", Source{}); err != nil {
		t.Fatalf("Add(secondary) error = %v", err)
	}

	registered := manager.Registered()
	if len(registered) != 2 {
		t.Fatalf("Registered() len = %d, want 2", len(registered))
	}

	registered[0] = "mutated"

	afterMutation := manager.Registered()
	if afterMutation[0] != "primary" {
		t.Fatalf("Registered() leaked internal slice, got %q want %q", afterMutation[0], "primary")
	}
}

func TestDBManagerGetInitializesExistingPlaceholder(t *testing.T) {
	driverName := registerDBManagerTestDriver(t)
	manager := &DBManager{
		sources: map[string]Source{
			"primary": {Driver: driverName},
		},
	}
	manager.conns.Store("primary", &conn{})

	db, drv, err := manager.Get("primary")
	if err != nil {
		t.Fatalf("Get(primary) error = %v", err)
	}
	if db == nil {
		t.Fatalf("Get(primary) returned nil db")
	}
	if drv == nil {
		t.Fatalf("Get(primary) returned nil driver")
	}
}

func TestDBManagerCloseIgnoresUninitializedPlaceholder(t *testing.T) {
	manager := &DBManager{}
	manager.conns.Store("primary", &conn{})

	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !manager.closed.Load() {
		t.Fatalf("Close() did not mark manager as closed")
	}
}

func TestDBManagerGetAfterCloseReturnsClosedError(t *testing.T) {
	driverName := registerDBManagerTestDriver(t)
	manager := &DBManager{
		sources: map[string]Source{
			"primary": {Driver: driverName},
		},
	}

	db, drv, err := manager.Get("primary")
	if err != nil {
		t.Fatalf("Get(primary) error = %v", err)
	}
	if db == nil || drv == nil {
		t.Fatalf("Get(primary) returned uninitialized connection")
	}

	if err := manager.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	db, drv, err = manager.Get("primary")
	if err != ErrDBManagerClosed {
		t.Fatalf("Get(primary) after Close() error = %v, want %v", err, ErrDBManagerClosed)
	}
	if db != nil || drv != nil {
		t.Fatalf("Get(primary) after Close() returned connection: db=%v drv=%v", db, drv)
	}
}

type invalidConfiguration struct{}

func (invalidConfiguration) Settings() SettingProvider { return keyValueSettingProvider{} }

func (invalidConfiguration) DefaultSource() string { return "" }

func (invalidConfiguration) Source(string) (Source, bool) { return Source{}, false }

func (invalidConfiguration) Sources() iter.Seq2[string, Source] {
	return func(func(string, Source) bool) {}
}

func (invalidConfiguration) Backend() configparser.Backend { return xmlparser.Backend{} }

func (invalidConfiguration) GetStatement(any) (Statement, error) { return nil, nil }

func (invalidConfiguration) Statement(StatementID) (Statement, error) { return nil, nil }

func TestNewDBManagerRejectsNilEnvironments(t *testing.T) {
	_, err := NewDBManager(nil)
	if !errors.Is(err, errConfigurationEnvironmentsRequired) {
		t.Fatalf("NewDBManager(nil) error = %v, want %v", err, errConfigurationEnvironmentsRequired)
	}
}

type inconsistentSourceProvider struct{}

func (inconsistentSourceProvider) DefaultSource() string { return "missing" }

func (inconsistentSourceProvider) Source(string) (Source, bool) { return Source{}, true }

func (inconsistentSourceProvider) Sources() iter.Seq2[string, Source] {
	return func(yield func(string, Source) bool) {
		yield("primary", Source{})
	}
}

func TestNewDBManagerValidatesDefaultAgainstSourceSnapshot(t *testing.T) {
	_, err := NewDBManager(inconsistentSourceProvider{})
	if !errors.Is(err, errConfigurationDefaultEnvironmentUnknown) {
		t.Fatalf("NewDBManager() error = %v, want %v", err, errConfigurationDefaultEnvironmentUnknown)
	}
}

type facetConfiguration struct {
	settings   SettingProvider
	backend    configparser.Backend
	statement  Statement
	runtime    *RuntimeConfig
	setCalls   int
	backCalls  int
	getCalls   int
	stateCalls int
}

func (c *facetConfiguration) Settings() SettingProvider {
	c.setCalls++
	return c.settings
}

func (c *facetConfiguration) DefaultSource() string {
	return c.runtime.DefaultSource()
}

func (c *facetConfiguration) Source(name string) (Source, bool) {
	return c.runtime.Source(name)
}

func (c *facetConfiguration) Sources() iter.Seq2[string, Source] {
	return c.runtime.Sources()
}

func (c *facetConfiguration) Backend() configparser.Backend {
	c.backCalls++
	return c.backend
}

func (c *facetConfiguration) GetStatement(any) (Statement, error) {
	c.getCalls++
	return c.statement, nil
}

func (c *facetConfiguration) Statement(StatementID) (Statement, error) {
	c.stateCalls++
	return c.statement, nil
}

func TestEngineUsesNarrowConfigurationFacets(t *testing.T) {
	driverName := registerDBManagerTestDriver(t)
	settings := keyValueSettingProvider{"debug": "false"}
	runtime, err := NewRuntimeConfig(
		"primary",
		map[string]Source{"primary": {Driver: driverName}},
		map[string]string{"debug": "false"},
	)
	if err != nil {
		t.Fatalf("NewRuntimeConfig() error = %v", err)
	}
	cfg := &facetConfiguration{
		settings:  settings,
		backend:   xmlparser.Backend{},
		statement: statementStub{},
		runtime:   runtime,
	}

	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = engine.Close() }()

	if cfg.setCalls != 1 || cfg.backCalls != 1 || cfg.getCalls != 0 || cfg.stateCalls != 0 {
		t.Fatalf("initial facet calls = settings:%d backend:%d get:%d statement:%d", cfg.setCalls, cfg.backCalls, cfg.getCalls, cfg.stateCalls)
	}

	if engine.Settings().Get("debug") != "false" {
		t.Fatal("engine did not retain settings facet")
	}
	if engine.Backend() != cfg.backend {
		t.Fatal("engine did not retain backend facet")
	}
	if statement := engine.Object("example.Statement").Statement(); statement == nil {
		t.Fatal("engine did not resolve statement through statement provider")
	}

	if cfg.setCalls != 1 || cfg.backCalls != 1 || cfg.getCalls != 0 || cfg.stateCalls != 1 {
		t.Fatalf("runtime facet calls = settings:%d backend:%d get:%d statement:%d", cfg.setCalls, cfg.backCalls, cfg.getCalls, cfg.stateCalls)
	}
}

func TestNewRejectsConfigurationWithoutEnvironments(t *testing.T) {
	_, err := New(invalidConfiguration{})
	if !errors.Is(err, errConfigurationEnvironmentsRequired) {
		t.Fatalf("New(invalidConfiguration{}) error = %v, want %v", err, errConfigurationEnvironmentsRequired)
	}
}
