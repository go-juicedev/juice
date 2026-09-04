package reflectlite

import (
	"reflect"
	"slices"
)

// IndirectType returns the underlying type if t is a pointer type.
// Otherwise, it returns t directly.
// For example, if t is *int, it returns int. If t is int, it returns int.
func IndirectType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// Type is a wrapper around reflect.Type that provides additional utility methods
// and caching for frequently accessed derived information like indirect type.
type Type struct {
	reflect.Type
	// indirectType holds the cached result of IndirectType(reflect.Type).
	// This avoids repeated computations if Indirect() is called multiple times.
	indirectType    reflect.Type
	indirectTypeSet bool // Tracks if indirectType has been computed and cached.
}

// Indirect returns a Type wrapper for the underlying type if the current type is a pointer.
// If the current type is not a pointer, it returns a Type wrapper for the current type itself.
// The result (the underlying reflect.Type) is cached within the Type wrapper for subsequent calls.
func (t *Type) Indirect() Type {
	if !t.indirectTypeSet {
		t.indirectType = IndirectType(t.Type) // Compute and cache
		t.indirectTypeSet = true
	}
	// Return a new Type wrapper around the (potentially cached) indirect reflect.Type.
	return Type{Type: t.indirectType, indirectType: t.indirectType, indirectTypeSet: true}
}

// getFieldIndexesFromTagRecursive is the internal recursive implementation for GetFieldIndexesFromTag.
// It searches for a field with the given tag name and value within the struct type.
// If found, it returns the field's index path and true. Otherwise, nil and false.
// It recursively searches embedded structs if the direct field does not have the tag
// or if the field is an anonymous struct.
func getFieldIndexesFromTagRecursive(typ reflect.Type, tagName, tagValue string) ([]int, bool) {
	for field := range typ.Fields() {
		// Check the tag on the current field.
		tag := field.Tag.Get(tagName)
		if tag == tagValue {
			return field.Index, true // Found the tag directly on this field.
		}

		// If the field is a struct, and it's either anonymous (embedded)
		// or it does not have the searched tag itself (meaning the tag might be in a sub-field of this struct field),
		// then recurse into this struct field.
		if field.Type.Kind() == reflect.Struct && (field.Anonymous || tag == "") {
			if indexes, ok := getFieldIndexesFromTagRecursive(field.Type, tagName, tagValue); ok {
				// Prepend current field's index to the indexes found in the nested struct.
				// This correctly builds the path to the tagged field.
				return slices.Concat(field.Index, indexes), true
			}
		}
	}
	return nil, false // Tag not found in this type or any of its relevant sub-structs.
}

// GetFieldIndexesFromTag searches for a field within the struct type `t` (or its underlying type if `t` is a pointer)
// that has a tag `tagName` with the value `tagValue`.
// It returns the field's index path (e.g., []int{0, 1} for a field nested in the first field) and true if found.
// Otherwise, it returns nil and false.
// Results are cached to improve performance on subsequent calls with the same type and tag criteria.
func (t *Type) GetFieldIndexesFromTag(tagName, tagValue string) ([]int, bool) {
	// Use the (cached) indirect type for all operations.
	indirect := t.Indirect() // This now correctly uses the pointer receiver and updates cache
	if indirect.Kind() != reflect.Struct {
		return nil, false
	}

	return getFieldIndexesFromTagRecursive(indirect.Type, tagName, tagValue)
}

// TypeFrom returns a new Type wrapper for the given reflect.Type.
// The indirect type is cached on the first call to Indirect().
func TypeFrom(t reflect.Type) *Type {
	return &Type{Type: t}
}
