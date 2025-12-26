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

package node

import (
	"fmt"
	"strings"

	"github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
)

// pureTextNode is a node of pure text.
// It is used to avoid unnecessary parameter replacement.
type pureTextNode string

func (p pureTextNode) Accept(_ driver.Translator, _ eval.Parameter) (query string, args []any, err error) {
	return string(p), nil, nil
}

func (p pureTextNode) AcceptTo(_ driver.Translator, _ eval.Parameter, builder *strings.Builder, _ *[]any) error {
	builder.WriteString(string(p))
	return nil
}

// pureTextNode is a node of pure text.
var _ NodeWriter = (*pureTextNode)(nil)

// TextNode is a node of text.
// What is the difference between TextNode and pureTextNode?
// TextNode is used to replace parameters with placeholders.
// pureTextNode is used to avoid unnecessary parameter replacement.
type TextNode struct {
	value            string
	placeholder      [][]string // for example, #{ID}
	textSubstitution [][]string // for example, ${ID}
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c *TextNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	// If there is no parameter, return the value as it is.
	if len(c.placeholder) == 0 && len(c.textSubstitution) == 0 {
		return c.value, nil, nil
	}
	// Otherwise, replace the parameter with a placeholder.
	query, args, err = c.replaceHolder(c.value, args, translator, p)
	if err != nil {
		return "", nil, err
	}
	query, err = c.replaceTextSubstitution(query, p)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

func (c *TextNode) replaceHolder(query string, args []any, translator driver.Translator, p eval.Parameter) (string, []any, error) {
	if len(c.placeholder) == 0 {
		return query, args, nil
	}

	builder := getStringBuilder()
	defer putStringBuilder(builder)
	builder.Grow(len(query))

	lastIndex := 0
	newArgs := make([]any, 0, len(args)+len(c.placeholder))
	newArgs = append(newArgs, args...)

	for _, param := range c.placeholder {
		if len(param) != 2 {
			return "", nil, fmt.Errorf("invalid parameter %v", param)
		}
		matched, name := param[0], param[1]

		value, exists := p.Get(name)
		if !exists {
			return "", nil, fmt.Errorf("parameter %s not found", name)
		}

		pos := strings.Index(query[lastIndex:], matched)
		if pos == -1 {
			continue
		}
		pos += lastIndex

		builder.WriteString(query[lastIndex:pos])
		builder.WriteString(translator.Translate(name))
		lastIndex = pos + len(matched)

		newArgs = append(newArgs, value.Interface())
	}

	builder.WriteString(query[lastIndex:])
	return builder.String(), newArgs, nil
}

// replaceTextSubstitution replaces text substitution.
func (c *TextNode) replaceTextSubstitution(query string, p eval.Parameter) (string, error) {
	if len(c.textSubstitution) == 0 {
		return query, nil
	}

	builder := getStringBuilder()
	defer putStringBuilder(builder)
	builder.Grow(len(query))

	lastIndex := 0
	for _, sub := range c.textSubstitution {
		if len(sub) != 2 {
			return "", fmt.Errorf("invalid text substitution %v", sub)
		}
		matched, name := sub[0], sub[1]

		value, exists := p.Get(name)
		if !exists {
			return "", fmt.Errorf("parameter %s not found", name)
		}

		pos := strings.Index(query[lastIndex:], matched)
		if pos == -1 {
			continue
		}
		pos += lastIndex

		builder.WriteString(query[lastIndex:pos])
		builder.WriteString(reflectValueToString(value))
		lastIndex = pos + len(matched)
	}

	builder.WriteString(query[lastIndex:])
	return builder.String(), nil
}

// NewTextNode creates a new text node based on the input string.
// It returns either a lightweight pureTextNode for static SQL,
// or a full TextNode for dynamic SQL with placeholders/substitutions.
func NewTextNode(str string) Node {
	placeholder := paramRegex.FindAllStringSubmatch(str, -1)

	textSubstitution := formatRegexp.FindAllStringSubmatch(str, -1)

	if len(placeholder) == 0 && len(textSubstitution) == 0 {
		return pureTextNode(str)
	}
	return &TextNode{value: str, placeholder: placeholder, textSubstitution: textSubstitution}
}

var _ Node = (*TextNode)(nil)
