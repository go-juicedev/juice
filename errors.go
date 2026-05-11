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
	"strings"

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

// nodeUnclosedError reports an unclosed XML node.
type nodeUnclosedError struct {
	nodeName string
	_        struct{}
}

// Error returns the error message.
func (e *nodeUnclosedError) Error() string {
	return fmt.Sprintf("node %s is not closed", e.nodeName)
}

// nodeAttributeRequiredError reports a missing required XML attribute.
type nodeAttributeRequiredError struct {
	nodeName string
	attrName string
}

// Error returns the error message.
func (e *nodeAttributeRequiredError) Error() string {
	return fmt.Sprintf("node %s requires attribute %s", e.nodeName, e.attrName)
}

// nodeAttributeConflictError reports conflicting XML attributes.
type nodeAttributeConflictError struct {
	nodeName string
	attrName string
}

// Error returns the error message.
func (e *nodeAttributeConflictError) Error() string {
	return fmt.Sprintf("node %s has conflicting attribute %s", e.nodeName, e.attrName)
}

// XMLParseError adds mapper and element context to an XML parsing error.
type XMLParseError struct {
	// Namespace is the namespace of the mapper being parsed
	Namespace string
	// XMLContent is the XML element content that caused the error
	XMLContent string
	// Err is the underlying error
	Err error
}

// Error returns the error message.
func (e *XMLParseError) Error() string {
	var builder strings.Builder
	builder.WriteString("XML parse error")
	if e.Namespace != "" {
		builder.WriteString(" in namespace '")
		builder.WriteString(e.Namespace)
		builder.WriteString("'")
	}
	if e.XMLContent != "" {
		builder.WriteString(": ")
		builder.WriteString(e.XMLContent)
	}
	if e.Err != nil {
		builder.WriteString(": ")
		builder.WriteString(e.Err.Error())
	}
	return builder.String()
}

// Unwrap returns the underlying error.
func (e *XMLParseError) Unwrap() error {
	return e.Err
}

// unreachable marks code paths that should never execute.
// nolint:deadcode,unused
func unreachable() error {
	panic("unreachable")
}
