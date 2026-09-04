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

// getFieldIndexesFromTagRecursive is the internal recursive implementation for LookupFieldIndexByTag.
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

// LookupFieldIndexByTag searches for a field within the struct type rType
// (or its underlying type if rType is a pointer) that has the tag tagName with
// the value tagValue. It returns the field's index path (e.g., []int{0, 1} for a
// field nested in the first field) and true if found, otherwise nil and false.
func LookupFieldIndexByTag(rType reflect.Type, tagName, tagValue string) ([]int, bool) {
	indirect := IndirectType(rType)
	if indirect == nil || indirect.Kind() != reflect.Struct {
		return nil, false
	}
	return getFieldIndexesFromTagRecursive(indirect, tagName, tagValue)
}
