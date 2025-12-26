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
	"sort"
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
	value  string
	tokens []textToken
}

type textToken struct {
	match    string
	name     string
	isFormat bool // true for ${...}, false for #{...}
	index    int
}

// Accept accepts parameters and returns query and arguments.
// Accept implements Node interface.
func (c *TextNode) Accept(translator driver.Translator, p eval.Parameter) (query string, args []any, err error) {
	// If there is no parameter, return the value as it is.
	if len(c.tokens) == 0 {
		return c.value, nil, nil
	}

	builder := getStringBuilder()
	defer putStringBuilder(builder)

	var capacity int
	for _, token := range c.tokens {
		if !token.isFormat {
			capacity++
		}
	}
	args = make([]any, 0, capacity)

	if err = c.AcceptTo(translator, p, builder, &args); err != nil {
		return "", nil, err
	}

	return builder.String(), args, nil
}

func (c *TextNode) AcceptTo(translator driver.Translator, p eval.Parameter, builder *strings.Builder, args *[]any) error {
	if len(c.tokens) == 0 {
		builder.WriteString(c.value)
		return nil
	}
	lastIndex := 0
	for _, t := range c.tokens {
		builder.WriteString(c.value[lastIndex:t.index])
		value, exists := p.Get(t.name)
		if !exists {
			return fmt.Errorf("parameter %s not found", t.name)
		}

		if t.isFormat {
			builder.WriteString(reflectValueToString(value))
		} else {
			builder.WriteString(translator.Translate(t.name))
			if args != nil {
				*args = append(*args, value.Interface())
			}
		}
		lastIndex = t.index + len(t.match)
	}
	builder.WriteString(c.value[lastIndex:])
	return nil
}

// NewTextNode creates a new text node based on the input string.
// It returns either a lightweight pureTextNode for static SQL,
// or a full TextNode for dynamic SQL with placeholders/substitutions.
func NewTextNode(str string) Node {
	placeholder := paramRegex.FindAllStringSubmatchIndex(str, -1)
	textSubstitution := formatRegexp.FindAllStringSubmatchIndex(str, -1)

	if len(placeholder) == 0 && len(textSubstitution) == 0 {
		return pureTextNode(str)
	}

	var tokens []textToken
	for _, p := range placeholder {
		tokens = append(tokens, textToken{
			match:    str[p[0]:p[1]],
			name:     str[p[2]:p[3]],
			isFormat: false,
			index:    p[0],
		})
	}
	for _, s := range textSubstitution {
		tokens = append(tokens, textToken{
			match:    str[s[0]:s[1]],
			name:     str[s[2]:s[3]],
			isFormat: true,
			index:    s[0],
		})
	}

	// Sort tokens by index
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].index < tokens[j].index
	})

	return &TextNode{value: str, tokens: tokens}
}

var _ NodeWriter = (*TextNode)(nil)
