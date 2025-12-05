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

// Unwrap continuously dereferences pointers and interfaces until a non-pointer/non-interface value is reached.
// If the initial value is not a pointer or interface, it's returned directly.
// This is useful for getting the underlying concrete value.
func Unwrap(value reflect.Value) reflect.Value {
	for value.IsValid() { // Ensure value is valid before checking Kind
		switch value.Kind() {
		case reflect.Ptr, reflect.Interface:
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
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return true
	default:
		return false
	}
}

// IndirectKind returns the Kind of the underlying type after dereferencing pointers.
// For example, if v is a reflect.Value of type *int, IndirectKind(v) returns reflect.Int.
func IndirectKind(v reflect.Value) reflect.Kind {
	// Use our own IndirectType which handles pointers, then get Kind.
	return IndirectType(v.Type()).Kind()
}

// Value is a wrapper around reflect.Value, providing utility methods.
type Value struct {
	reflect.Value
}

// Unwrap returns a Value wrapper for the underlying concrete value,
// after continuously dereferencing pointers and interfaces.
func (v *Value) Unwrap() Value {
	return Value{Value: Unwrap(v.Value)}
}

// IsNilable checks if the wrapped reflect.Value can be nil.
// See the global IsNilable function for more details.
func (v Value) IsNilable() bool {
	return IsNilable(v.Value)
}

// IndirectType returns a Type wrapper for the underlying type of the wrapped reflect.Value,
// after dereferencing pointers.
func (v *Value) IndirectType() Type {
	underlyingT := IndirectType(v.Type())
	return *TypeFrom(underlyingT)
}

// IndirectKind returns the Kind of the underlying type of the wrapped reflect.Value,
// after dereferencing pointers.
func (v *Value) IndirectKind() reflect.Kind {
	// Leverage the cached Type wrapper if available.
	return v.IndirectType().Kind()
}

// findFieldFromTagRecursive is the internal recursive implementation for FindFieldFromTag.
// It searches for a field with the given tag name and value within the Value's type.
// If found, it returns a Value wrapper for that field and true. Otherwise, an invalid Value and false.
// It recursively searches embedded or nested structs.
func findFieldFromTagRecursive(val *Value, tagName, tagValue string) (*Value, bool) {
	// Work with the indirect type of the current value.
	valType := val.IndirectType() // Uses the cached Type wrapper from Value
	if valType.Kind() != reflect.Struct {
		return nil, false
	}

	// Iterate through the fields of the struct.
	numFields := valType.NumField()
	for i := 0; i < numFields; i++ {
		field := valType.Field(i)         // This is a reflect.StructField
		fieldVal := val.Unwrap().Field(i) // This is a reflect.Value for the field

		// Check the tag on the current field.
		if tag := field.Tag.Get(tagName); tag == tagValue {
			return ValueFrom(fieldVal), true
		}

		// If the field is a struct and it's either anonymous (embedded) or does not have the searched tag,
		// recurse into this struct field.
		if field.Type.Kind() == reflect.Struct && (field.Anonymous || field.Tag.Get(tagName) == "") {
			// Pass a Value wrapper for the field to the recursive call.
			if v, ok := findFieldFromTagRecursive(ValueFrom(fieldVal), tagName, tagValue); ok {
				return v, true
			}
		}
	}
	return nil, false // Tag not found.
}

// FindFieldFromTag searches for a field within the Value's underlying struct type
// (after dereferencing pointers/interfaces) that has a tag `tagName` with the value `tagValue`.
// It returns a Value wrapper for the field if found, and true. Otherwise, it returns an invalid Value and false.
// Note: Caching for this function is not implemented here but could be added similarly to GetFieldIndexesFromTag in Type,
// using a combination of the Value's type, tagName, and tagValue as the key.
func (v *Value) FindFieldFromTag(tagName, tagValue string) (*Value, bool) {
	// Ensure we are operating on a struct or a pointer to a struct.
	// The recursive helper will handle the unwrapping.
	if v.Unwrap().Kind() != reflect.Struct {
		return nil, false
	}
	// TODO: Consider adding caching for FindFieldFromTag if it becomes a performance bottleneck.
	// The cache key would likely involve v.IndirectType().Type, tagName, and tagValue.
	return findFieldFromTagRecursive(v, tagName, tagValue)
}

// GetFieldIndexesFromTag returns the field indexes by tag name and tag value,
// by delegating to the cached Type wrapper.
func (v *Value) GetFieldIndexesFromTag(tagName, tagValue string) ([]int, bool) {
	// Use the IndirectType method which initializes and/or returns the cached Type wrapper.
	// Then call GetFieldIndexesFromTag on that Type wrapper.
	typeWrapper := v.IndirectType() // This ensures typeWrapper is initialized
	return typeWrapper.GetFieldIndexesFromTag(tagName, tagValue)
}

// ValueOf returns a new Value wrapper initialized to the concrete value
// stored in the interface i. ValueOf(nil) returns the zero Value.
func ValueOf(v any) *Value {
	return &Value{Value: reflect.ValueOf(v)} // Explicitly name the field
}

// ValueFrom returns a new Value initialized to the concrete value
func ValueFrom(v reflect.Value) *Value {
	return &Value{Value: v} // Explicitly name the field
}
