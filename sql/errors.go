/*
Copyright 2025 eatmoreapple

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

package sql

import "errors"

var (
	// ErrNilDestination is an error that is returned when the destination is nil.
	ErrNilDestination = errors.New("destination can not be nil")

	// ErrNilRows is an error that is returned when the rows is nil.
	ErrNilRows = errors.New("rows can not be nil")

	// ErrResultMapNotSet is an error that is returned when the result map is not set.
	ErrResultMapNotSet = errors.New("resultMap not set")

	// ErrPointerRequired is an error that is returned when the destination is not a pointer.
	ErrPointerRequired = errors.New("destination must be a pointer")
)
