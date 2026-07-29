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
	"os"
	"sync"
)

// EnvValueProvider defines a environment value provider.
type EnvValueProvider interface {
	Get(key string) (string, error)
}

// EnvValueProviderLookup resolves a named environment value provider.
type EnvValueProviderLookup func(name string) (EnvValueProvider, bool)

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
