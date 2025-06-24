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

package expr

import (
	"reflect"

	"github.com/go-juicedev/juice/internal/reflectlite"
)

func isInt(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func isAllInt(rs ...reflect.Value) bool {
	if len(rs) == 0 {
		return false
	}
	for _, r := range rs {
		if !isInt(r) {
			return false
		}
	}
	return true
}

func isUint(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isAllUint(rs ...reflect.Value) bool {
	if len(rs) == 0 {
		return false
	}
	for _, r := range rs {
		if !isUint(r) {
			return false
		}
	}
	return true
}

func isFloat(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isAllFloat(rs ...reflect.Value) bool {
	if len(rs) == 0 {
		return false
	}
	for _, r := range rs {
		if !isFloat(r) {
			return false
		}
	}
	return true
}

func isComplex(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

func isAllComplex(rs ...reflect.Value) bool {
	if len(rs) == 0 {
		return false
	}
	for _, r := range rs {
		if !isComplex(r) {
			return false
		}
	}
	return true
}

func isString(r reflect.Value) bool {
	return r.Kind() == reflect.String
}

func isAllString(rs ...reflect.Value) bool {
	if len(rs) == 0 {
		return false
	}
	for _, r := range rs {
		if !isString(r) {
			return false
		}
	}
	return true
}

func isBool(r reflect.Value) bool {
	return r.Kind() == reflect.Bool
}

func isAllBool(rs ...reflect.Value) bool {
	if len(rs) == 0 {
		return false
	}
	for _, r := range rs {
		if !isBool(r) {
			return false
		}
	}
	return true
}

func bothNil(left, right reflect.Value) bool {
	if !right.IsValid() || !left.IsValid() {

		// if both values are invalid, they are equal
		if !right.IsValid() && !left.IsValid() {
			return true
		}
		var valid = right
		if !right.IsValid() {
			valid = left
		}

		// if the invalid value is nil, the valid value is equal to nil
		if reflectlite.IsNilable(valid) {
			// nil value
			if valid.Equal(nilValue) {
				return true
			}

			// unwrap interface value
			if valid.Kind() == reflect.Interface {
				valid = valid.Elem()
			}
			return valid.IsNil()
		}
	}
	return false
}
