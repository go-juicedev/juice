package reflectlite

import "reflect"

// Unpack recursively unwraps interface values until it reaches a non-interface value.
// For nil interfaces, it returns the original value.
// For non-interface values, it returns the value as is.
func Unpack(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	for v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	return v
}
