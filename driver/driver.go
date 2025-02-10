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

package driver

import (
	"fmt"
	"maps"
	"slices"
	"sync"
)

// Driver is a driver of database.
type Driver interface {
	// Translator returns a translator of SQL.
	Translator() Translator
}

var (
	// registeredDrivers is a map of registered drivers.
	// The key is a name of driver, it is used to get a driver.
	registeredDrivers = make(map[string]Driver)

	// lock is a lock for registeredDrivers.
	// For thread safety.
	driversMu sync.RWMutex
)

// Register registers a driver.
// The name is used to get a driver.
func Register(name string, driver Driver) {
	if driver == nil {
		panic("driver: Register driver is nil")
	}
	driversMu.Lock()
	defer driversMu.Unlock()
	// allow re-registration
	registeredDrivers[name] = driver
}

// Get returns a driver of the name.
// If the name is not registered, it returns an error.
func Get(name string) (Driver, error) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	driver, ok := registeredDrivers[name]
	if !ok {
		return nil, fmt.Errorf("driver %s not found", name)
	}
	return driver, nil
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	return slices.Sorted(maps.Keys(registeredDrivers))
}
