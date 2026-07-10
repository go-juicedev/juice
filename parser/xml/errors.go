/*
Copyright 2026 eatmoreapple

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

package xml

import "fmt"

type ParseError struct {
	Element string
	Err     error
}

func (e *ParseError) Error() string {
	if e.Element == "" {
		return fmt.Sprintf("xml parse error: %v", e.Err)
	}
	return fmt.Sprintf("xml parse error in <%s>: %v", e.Element, e.Err)
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

func wrap(element string, err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*ParseError); ok {
		return err
	}
	return &ParseError{Element: element, Err: err}
}
