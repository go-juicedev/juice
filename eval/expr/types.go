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

func isIntType(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func isUintType(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloatType(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

func isComplexType(r reflect.Value) bool {
	switch r.Kind() {
	case reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

func isNumeric(r reflect.Value) bool {
	return isIntType(r) || isUintType(r) || isFloatType(r) || isComplexType(r)
}

func isStringType(r reflect.Value) bool {
	return r.Kind() == reflect.String
}

func isBoolType(r reflect.Value) bool {
	return r.Kind() == reflect.Bool
}

func allOf(predicate func(reflect.Value) bool, values ...reflect.Value) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !predicate(value) {
			return false
		}
	}
	return true
}

func anyOf(predicate func(reflect.Value) bool, values ...reflect.Value) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if predicate(value) {
			return true
		}
	}
	return false
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
