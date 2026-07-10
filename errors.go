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

package juice

import (
	"errors"

	"github.com/go-juicedev/juice/sql"
)

var (
	// ErrEmptyQuery is returned when the rendered query is empty.
	ErrEmptyQuery = errors.New("empty query")

	// ErrPointerRequired is returned when the destination is not a pointer.
	ErrPointerRequired = sql.ErrPointerRequired

	// errSliceOrArrayRequired is returned when the destination is not a slice or array.
	errSliceOrArrayRequired = errors.New("type must be a slice or array")

	// ErrNoStatementFound is returned when no mapped statement exists.
	ErrNoStatementFound = errors.New("no statement found")

	// ErrNoManagerFoundInContext is returned when the context has no manager.
	ErrNoManagerFoundInContext = errors.New("no manager found in context")
)
