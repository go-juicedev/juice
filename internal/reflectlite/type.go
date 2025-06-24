package reflectlite

import (
	"reflect"
	"strings"
	"sync"
)

// cache for field indexes
var fieldIndexCache = &sync.Map{}

// cacheKey is used as a key for caching field indexes.
type cacheKey struct {
	Type     reflect.Type
	TagName  string
	TagValue string
}

// IndirectType returns the underlying type if t is a pointer type.
// Otherwise, it returns t directly.
// For example, if t is *int, it returns int. If t is int, it returns int.
func IndirectType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// typeToString returns a string representation of the reflect.Type, including
// the package path for non-built-in types. It uses strings.Builder for efficient concatenation.
func typeToString(t reflect.Type) string {
	var sb strings.Builder
	writeTypeString(&sb, t)
	return sb.String()
}

// writeTypeString is a helper recursive function for typeToString.
func writeTypeString(sb *strings.Builder, t reflect.Type) {
	if t == nil {
		sb.WriteString("<nil>")
		return
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Ptr, reflect.Chan:
		sb.WriteString(t.Kind().String())
		sb.WriteString("[")
		writeTypeString(sb, t.Elem())
		sb.WriteString("]")
	case reflect.Map:
		sb.WriteString("map[")
		writeTypeString(sb, t.Key())
		sb.WriteString("]")
		writeTypeString(sb, t.Elem())
	case reflect.Struct, reflect.Interface:
		if t.Name() == "" {
			// This is an anonymous struct or interface.
			sb.WriteString(t.String()) // Fallback to default String() for anonymous complex types
		} else {
			// For named struct and interface types, include the package path.
			sb.WriteString(qualifiedName(t))
		}
	default:
		// For other types (including basic types and named types).
		sb.WriteString(qualifiedName(t))
	}
}

// qualifiedName returns the name of the type with its package path if it's not a built-in type.
// Example: "main.MyStruct" or "int".
func qualifiedName(t reflect.Type) string {
	if t.PkgPath() != "" && t.Name() != "" {
		return t.PkgPath() + "." + t.Name()
	}
	// For built-in types or unnamed types, t.String() is usually sufficient.
	return t.String()
}

// TypeIdentify returns a string representation of the type T, including the package path for non-built-in types.
// This is useful for generating unique identifiers for types.
func TypeIdentify[T any]() string {
	// Use reflect.TypeOf((*T)(nil)).Elem() to get the type of T itself,
	// as reflect.TypeOf(T) would result in "reflect.rtype" if T is a type.
	return typeToString(reflect.TypeOf((*T)(nil)).Elem())
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

// Identify returns a string representation of the wrapped reflect.Type,
// including the package path for non-built-in types.
// This is useful for logging or generating type-specific identifiers.
func (t Type) Identify() string {
	return typeToString(t.Type)
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
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		// Check the tag on the current field.
		if tag := field.Tag.Get(tagName); tag == tagValue {
			return field.Index, true // Found the tag directly on this field.
		}

		// If the field is a struct, and it's either anonymous (embedded)
		// or it does not have the searched tag itself (meaning the tag might be in a sub-field of this struct field),
		// then recurse into this struct field.
		if field.Type.Kind() == reflect.Struct && (field.Anonymous || field.Tag.Get(tagName) == "") {
			if indexes, ok := getFieldIndexesFromTagRecursive(field.Type, tagName, tagValue); ok {
				// Prepend current field's index to the indexes found in the nested struct.
				// This correctly builds the path to the tagged field.
				return append(field.Index, indexes...), true
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

	key := cacheKey{Type: indirect.Type, TagName: tagName, TagValue: tagValue}
	if cached, ok := fieldIndexCache.Load(key); ok {
		// Type assertion to the specific structure used for caching.
		if entry, valid := cached.(struct {
			indexes []int
			found   bool
		}); valid {
			return entry.indexes, entry.found
		}
	}

	indexes, found := getFieldIndexesFromTagRecursive(indirect.Type, tagName, tagValue)

	// Cache the actual result (indexes and found status).
	fieldIndexCache.Store(key, struct {
		indexes []int
		found   bool
	}{indexes: indexes, found: found})

	return indexes, found
}

// TypeFrom returns a new Type wrapper for the given reflect.Type.
// The indirect type is not yet cached at this point; it will be on the first call to Indirect().
func TypeFrom(t reflect.Type) Type {
	return Type{Type: t}
}
