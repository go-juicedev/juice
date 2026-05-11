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
	"encoding"
	"strconv"
)

// StringValue is a string that can be converted to common scalar types.
type StringValue string

// Bool returns true if the value is "true".
func (s StringValue) Bool() bool {
	value, _ := strconv.ParseBool(string(s))
	return value
}

// Int64 returns the value as int64.
func (s StringValue) Int64() int64 {
	value, _ := strconv.ParseInt(string(s), 10, 64)
	return value
}

// Uint64 returns the value as uint64.
func (s StringValue) Uint64() uint64 {
	value, _ := strconv.ParseUint(string(s), 10, 64)
	return value
}

// String returns the value as string.
func (s StringValue) String() string {
	return string(s)
}

// Float64 returns the value as float64.
func (s StringValue) Float64() float64 {
	value, _ := strconv.ParseFloat(string(s), 64)
	return value
}

// Unmarshaler unmarshals the value into marshaller.
func (s StringValue) Unmarshaler(marshaller encoding.TextUnmarshaler) error {
	return marshaller.UnmarshalText([]byte(s))
}

type SettingProvider interface {
	Get(name string) StringValue
}

// keyValueSettingProvider is a collection of settings.
type keyValueSettingProvider map[string]StringValue

// Get returns the value of the key.
func (s keyValueSettingProvider) Get(name string) StringValue {
	return s[name]
}

// ensure keyValueSettingProvider implements SettingProvider.
var _ SettingProvider = (*keyValueSettingProvider)(nil)

// settingItem is a setting element.
type settingItem struct {
	// The name of the setting.
	Name string `xml:"name,attr"`
	// The value of the setting.
	Value StringValue `xml:"value,attr"`
}
