package juice

import (
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	jdriver "github.com/go-juicedev/juice/driver"
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
