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

package eval

import (
	"errors"
	"reflect"
	"strings"
)

// return the length of the string or array
func length(v any) (int, error) {
	switch t := v.(type) {
	case nil:
		return 0, nil
	case string:
		return len(t), nil
	default:
		rv := reflect.Indirect(reflect.ValueOf(v))
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
			return rv.Len(), nil
		default:
		}
	}
	return 0, errors.New("length: invalid argument type")
}

// strSub returns a substring of the string.
// The first parameter is the string to be processed.
// The second parameter is the start position of the substring.
// The third parameter is the length of the substring.
func strSub(str string, start, count int) (string, error) {
	if start < 0 {
		start = len(str) + start
	}
	if start < 0 {
		start = 0
	}
	if start > len(str) {
		start = len(str)
	}
	if count < 0 {
		count = len(str) + count
	}
	if count < 0 {
		count = 0
	}
	if start+count > len(str) {
		count = len(str) - start
	}
	return str[start : start+count], nil
}

// strJoin joins the elements of the array into a string.
// The first parameter is the array to be processed.
// The second parameter is the separator.
// Returns a string.
func strJoin(v any, sep string) (string, error) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.String {
			list := make([]string, 0, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				list = append(list, rv.Index(i).String())
			}
			return strings.Join(list, sep), nil
		}
	default:
	}
	return "", errors.New("join: invalid argument type")
}

// slice returns a slice of the array or string.
func slice(v any, start, count int) ([]any, error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	switch rv.Kind() {
	case reflect.Array, reflect.Slice:
		rt := rv.Slice(start, start+count)
		var ret = make([]any, 0, rt.Len())
		for i := 0; i < rt.Len(); i++ {
			ret = append(ret, rt.Index(i).Interface())
		}
		return ret, nil
	default:
	}
	return nil, errors.New("slice: invalid argument type")
}

// lower returns a copy of the string s with all Unicode letters mapped to their lower case.
func lower(text string) (string, error) {
	return strings.ToLower(text), nil
}

// upper returns a copy of the string s with all Unicode letters mapped to their upper case.
func upper(text string) (string, error) {
	return strings.ToUpper(text), nil
}

// trim returns a slice of the string s with all leading and trailing Unicode code points contained in cutset removed.
func trim(text, cutest string) (string, error) {
	return strings.Trim(text, cutest), nil
}

// trimLeft returns a slice of the string s with all leading Unicode code points contained in cutset removed.
func trimLeft(text, cutest string) (string, error) {
	return strings.TrimLeft(text, cutest), nil
}

// trimRight returns a slice of the string s with all trailing Unicode code points contained in cutset removed.
func trimRight(text, cutest string) (string, error) {
	return strings.TrimRight(text, cutest), nil
}

// replace returns a copy of the string s with the first n non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune string.
func replace(text, old, new string, n int) (string, error) {
	return strings.Replace(text, old, new, n), nil
}

// replaceAll returns a copy of the string s with all non-overlapping instances of old replaced by new.
// If old is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune string.
func replaceAll(text, old, new string) (string, error) {
	return strings.ReplaceAll(text, old, new), nil
}

// split returns a slice of strings after splitting the string s at each instance of sep.
func split(text, sep string) ([]string, error) {
	return strings.Split(text, sep), nil
}

// splitN returns a slice of strings after splitting the string s at each instance of sep, at most n times.
// If n == 0, SplitN returns an unlimited number of substrings.
// If n < 0, SplitN splits after each instance of sep.
func splitN(text, sep string, n int) ([]string, error) {
	return strings.SplitN(text, sep, n), nil
}

// splitAfter returns a slice of strings after splitting the string s after each instance of sep.
func splitAfter(text, sep string) ([]string, error) {
	return strings.SplitAfter(text, sep), nil
}

// RegisterEvalFunc registers a function for eval.
// The function must be a function with one return value.
// And Allowed to overwrite the built-in function.
func RegisterEvalFunc(name string, v any) error {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if rv.Kind() != reflect.Func {
		return errors.New("RegisterEvalFunc: v must be a function type")
	}
	if rv.Type().NumOut() != 2 {
		return errors.New("RegisterEvalFunc: v must be a function with two return value")
	}
	// if last return value is error
	if !rv.Type().Out(rv.Type().NumOut() - 1).Implements(errType) {
		return errors.New("RegisterEvalFunc: v must be a function with an error return value")
	}
	builtins[name] = rv
	return nil
}

// MustRegisterEvalFunc registers a function for eval.
// If an error occurs, it will panic.
func MustRegisterEvalFunc(name string, v any) {
	err := RegisterEvalFunc(name, v)
	if err != nil {
		panic(err)
	}
}

// errType is the reflect.Type of error.
var errType = reflect.TypeOf((*error)(nil)).Elem()

// builtins is a map of built-in functions.
var builtins = map[string]reflect.Value{}

var (
	// trueValue is the reflect.Value of true.
	trueValue = reflect.ValueOf(true)

	// falseValue is the reflect.Value of false.
	falseValue = reflect.ValueOf(false)

	// nilValue is the reflect.Value of nil.
	nilValue = reflect.ValueOf(nil)
)

func init() {
	builtins["true"] = trueValue
	builtins["false"] = falseValue
	builtins["nil"] = nilValue
	MustRegisterEvalFunc("len", length)
	MustRegisterEvalFunc("substr", strSub)
	MustRegisterEvalFunc("join", strJoin)
	MustRegisterEvalFunc("slice", slice)
	MustRegisterEvalFunc("lower", lower)
	MustRegisterEvalFunc("upper", upper)
	MustRegisterEvalFunc("trim", trim)
	MustRegisterEvalFunc("trimLeft", trimLeft)
	MustRegisterEvalFunc("trimRight", trimRight)
	MustRegisterEvalFunc("replace", replace)
	MustRegisterEvalFunc("replaceAll", replaceAll)
	MustRegisterEvalFunc("split", split)
	MustRegisterEvalFunc("splitN", splitN)
	MustRegisterEvalFunc("splitAfter", splitAfter)
}
