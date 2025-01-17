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
	"fmt"
)

var (
	// ErrEmptyQuery is an error that is returned when the query is empty.
	ErrEmptyQuery = errors.New("empty query")

	// ErrResultMapNotSet is an error that is returned when the result map is not set.
	ErrResultMapNotSet = errors.New("resultMap not set")

	// ErrSqlNodeNotFound is an error that is returned when the sql node is not found.
	// nolint:unused
	ErrSqlNodeNotFound = errors.New("sql node not found")

	// ErrNilDestination is an error that is returned when the destination is nil.
	ErrNilDestination = errors.New("destination can not be nil")

	// ErrNilRows is an error that is returned when the rows is nil.
	ErrNilRows = errors.New("rows can not be nil")

	// ErrPointerRequired is an error that is returned when the destination is not a pointer.
	ErrPointerRequired = errors.New("destination must be a pointer")

	// errSliceOrArrayRequired is an error that is returned when the destination is not a slice or array.
	errSliceOrArrayRequired = errors.New("type must be a slice or array")

	// ErrNoStatementFound is an error that is returned when the statement is not found.
	ErrNoStatementFound = errors.New("no statement found")
)

// nodeUnclosedError is an error that is returned when the node is not closed.
type nodeUnclosedError struct {
	nodeName string
	_        struct{}
}

// Error returns the error message.
func (e *nodeUnclosedError) Error() string {
	return fmt.Sprintf("node %s is not closed", e.nodeName)
}

// nodeAttributeRequiredError is an error that is returned when the node requires an attribute.
type nodeAttributeRequiredError struct {
	nodeName string
	attrName string
}

// Error returns the error message.
func (e *nodeAttributeRequiredError) Error() string {
	return fmt.Sprintf("node %s requires attribute %s", e.nodeName, e.attrName)
}

// nodeAttributeConflictError is an error that is returned when the node has conflicting attributes.
type nodeAttributeConflictError struct {
	nodeName string
	attrName string
}

// Error returns the error message.
func (e *nodeAttributeConflictError) Error() string {
	return fmt.Sprintf("node %s has conflicting attribute %s", e.nodeName, e.attrName)
}

// unreachable is a function that is used to mark unreachable code.
// nolint:deadcode,unused
func unreachable() error {
	panic("unreachable")
}

// ErrInvalidStatementID indicates that the statement ID format is invalid
var ErrInvalidStatementID = errors.New("invalid statement id: must be in format namespace.statementName")

// ErrMapperNotFound indicates that the mapper was not found
type ErrMapperNotFound string

func (e ErrMapperNotFound) Error() string { return fmt.Sprintf("mapper %q not found", string(e)) }

// ErrStatementNotFound indicates that the statement was not found in the mapper
type ErrStatementNotFound struct {
	StatementName string
	MapperName    string
}

func (e ErrStatementNotFound) Error() string {
	return fmt.Sprintf("statement %q not found in mapper %q", e.StatementName, e.MapperName)
}
