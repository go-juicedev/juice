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
	"iter"
	"maps"
	"os"
	"sync"
)

// Environment defines a environment.
// It contains a database connection configuration.
type Environment struct {
	// DataSource is a string in a driver-specific format.
	DataSource string

	// Driver is a driver for
	Driver string

	// MaxIdleConnNum is a maximum number of idle connections.
	MaxIdleConnNum int

	// MaxOpenConnNum is a maximum number of open connections.
	MaxOpenConnNum int

	// MaxConnLifetime is a maximum lifetime of a connection.
	MaxConnLifetime int

	// MaxIdleConnLifetime is a maximum lifetime of an idle connection.
	MaxIdleConnLifetime int

	// attrs is a map of attributes.
	attrs map[string]string
}

// setAttr sets a value of the attribute.
func (e *Environment) setAttr(key, value string) {
	if e.attrs == nil {
		e.attrs = make(map[string]string)
	}
	e.attrs[key] = value
}

// Attr returns a value of the attribute.
func (e *Environment) Attr(key string) string {
	return e.attrs[key]
}

// ID returns an identifier of the environment.
func (e *Environment) ID() string {
	return e.Attr("id")
}

// provider is a environment value provider.
// It provides a value of the environment variable.
func (e *Environment) provider() (EnvValueProvider, error) {
	name := e.Attr("provider")
	provider, ok := LookupEnvValueProvider(name)
	if ok {
		return provider, nil
	}
	return nil, fmt.Errorf("%w: %s", errEnvValueProviderNotFound, name)
}

type EnvironmentProvider interface {
	// Attribute returns a value of the attribute.
	Attribute(key string) string

	// Use returns the environment specified by the identifier.
	Use(id string) (*Environment, error)

	Iter() iter.Seq2[string, *Environment]
}

// environments is a collection of environments.
type environments struct {
	// xml attributes stored as key-value pairs.
	attr map[string]string

	// envs is a map of environments.
	// The key is an identifier of the environment.
	envs map[string]*Environment
}

// setAttr sets a value of the attribute.
func (e *environments) setAttr(key, value string) {
	if e.attr == nil {
		e.attr = make(map[string]string)
	}
	e.attr[key] = value
}

// Attribute returns a value of the attribute.
func (e *environments) Attribute(key string) string {
	return e.attr[key]
}

// Use returns the environment specified by the identifier.
func (e *environments) Use(id string) (*Environment, error) {
	env, exists := e.envs[id]
	if !exists {
		return nil, fmt.Errorf("environment %s not found", id)
	}
	return env, nil
}

// Iter returns a sequence of environments.
func (e *environments) Iter() iter.Seq2[string, *Environment] {
	return maps.All(e.envs)
}

// EnvValueProvider defines a environment value provider.
type EnvValueProvider interface {
	Get(key string) (string, error)
}

var (
	// envValueProviderLibraries is a map of environment value providers.
	envValueProviderLibraries = map[string]EnvValueProvider{}

	// envValueProviderMu protects envValueProviderLibraries.
	envValueProviderMu sync.RWMutex
)

// EnvValueProviderFunc is a function type of environment value provider.
type EnvValueProviderFunc func(key string) (string, error)

// Get is a function type of environment value provider.
func (f EnvValueProviderFunc) Get(key string) (string, error) {
	return f(key)
}

// OsEnvValueProvider is a environment value provider that uses os.Expand.
type OsEnvValueProvider struct{}

// Get returns a value of the environment variable.
// It uses os.Expand and follows the standard library expansion rules.
func (p OsEnvValueProvider) Get(key string) (string, error) {
	return os.Expand(key, os.Getenv), nil
}

var (
	errEnvValueProviderNameEmpty = errors.New("name is empty")
	errEnvValueProviderNil       = errors.New("juice: environment value provider is nil")
	errEnvValueProviderNotFound  = errors.New("juice: environment value provider not found")
)

// RegisterEnvValueProvider registers an environment value provider.
// The key is a name of the provider.
// The value is a provider.
// It allows to override the default provider.
func RegisterEnvValueProvider(name string, provider EnvValueProvider) error {
	if len(name) == 0 {
		return errEnvValueProviderNameEmpty
	}
	if provider == nil {
		return errEnvValueProviderNil
	}
	envValueProviderMu.Lock()
	defer envValueProviderMu.Unlock()
	envValueProviderLibraries[name] = provider
	return nil
}

// MustRegisterEnvValueProvider registers an environment value provider and panics on error.
func MustRegisterEnvValueProvider(name string, provider EnvValueProvider) {
	if err := RegisterEnvValueProvider(name, provider); err != nil {
		panic(err)
	}
}

// passthroughEnvValueProvider returns the key unchanged and never errors.
// This is the default provider when no named provider is registered.
type passthroughEnvValueProvider struct{}

func (passthroughEnvValueProvider) Get(key string) (string, error) {
	return key, nil
}

// LookupEnvValueProvider returns a provider and whether it was found.
// An empty key uses the passthrough provider and reports success.
func LookupEnvValueProvider(key string) (EnvValueProvider, bool) {
	if len(key) == 0 {
		return passthroughEnvValueProvider{}, true
	}

	envValueProviderMu.RLock()
	defer envValueProviderMu.RUnlock()
	if provider, exists := envValueProviderLibraries[key]; exists {
		return provider, true
	}
	return nil, false
}

func init() {
	// Register the default environment value provider.
	MustRegisterEnvValueProvider("env", &OsEnvValueProvider{})
}
