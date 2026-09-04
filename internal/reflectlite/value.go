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

package reflectlite

import "reflect"

// FromInt returns a reflect.Value for the given int, using a cache for small integers (0-255).
func FromInt(i int) reflect.Value {
	if i >= 0 && i < 256 {
		return intCache[i]
	}
	return reflect.ValueOf(i)
}

var intCache [256]reflect.Value

func init() {
	for i := range 256 {
		intCache[i] = reflect.ValueOf(i)
	}
}

// Unwrap continuously dereferences pointers and interfaces until a non-pointer/non-interface value is reached.
// If the initial value is not a pointer or interface, it's returned directly.
// This is useful for getting the underlying concrete value.
func Unwrap(value reflect.Value) reflect.Value {
	for value.IsValid() { // Ensure value is valid before checking Kind
		switch value.Kind() {
		case reflect.Pointer, reflect.Interface:
			if value.IsNil() { // Stop if we encounter a nil pointer/interface
				return value
			}
			value = value.Elem()
		default:
			return value
		}
	}
	return value // Return original invalid value if that was passed
}

// IsNilable checks if a reflect.Value can be nil.
// This includes channels, functions, interfaces, maps, pointers, slices, and unsafe pointers.
// It also correctly handles an invalid reflect.Value, returning true as it represents a "nil-like" state.
func IsNilable(v reflect.Value) bool {
	if !v.IsValid() { // An invalid reflect.Value is effectively nil.
		return true
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return true
	default:
		return false
	}
}

// LookupFieldByTag searches for a field within the struct value v (after
// dereferencing pointers/interfaces) that has the tag tagName with the value
// tagValue. It returns the field's reflect.Value and true if found, otherwise
// an invalid reflect.Value and false.
func LookupFieldByTag(v reflect.Value, tagName, tagValue string) (reflect.Value, bool) {
	v = Unwrap(v)
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	indexes, ok := LookupFieldIndexByTag(v.Type(), tagName, tagValue)
	if !ok {
		return reflect.Value{}, false
	}
	return v.FieldByIndex(indexes), true
}
