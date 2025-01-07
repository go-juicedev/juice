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

// Package eval provides a simple lexical analyzer for processing logical expressions.
// It converts human-readable logical operators (and, or, not) to their Go equivalents (&&, ||, !).
package eval

import (
	"go/scanner"
	"go/token"
	"strings"
)

// identReplacer converts logical operators from human-readable format to Go syntax.
// It maps:
//   - "and" to "&&"
//   - "or" to "||"
//   - "not" to "!"
//
// Any other identifiers are returned unchanged.
func identReplacer(s string) string {
	switch s {
	case "and":
		return "&&"
	case "or":
		return "||"
	case "not":
		return "!"
	default:
		return s
	}
}

// Lexer performs lexical analysis on input strings.
// It uses Go's standard scanner to tokenize the input and processes
// specific identifiers for logical operations.
type Lexer struct {
	scanner scanner.Scanner
}

// Tokenize processes the input and returns a string with converted operators.
// It scans through all tokens, replacing logical operators while preserving
// other tokens and maintaining proper spacing.
func (l *Lexer) Tokenize() string {
	var tokens []string
	for {
		_, tok, lit := l.scanner.Scan()
		if tok == token.EOF {
			break
		}

		switch tok {
		case token.IDENT:
			replacement := identReplacer(lit)
			tokens = append(tokens, replacement)
		default:
			if lit != "" {
				tokens = append(tokens, lit)
			} else {
				tokens = append(tokens, tok.String())
			}
		}
	}

	return strings.Join(tokens, " ")
}

// NewLexer creates a new Lexer instance with the given input string.
// It initializes the internal scanner with the input and configures it
// to scan comments as well.
func NewLexer(input string) *Lexer {
	var s scanner.Scanner
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(input))

	s.Init(file, []byte(input), nil, scanner.ScanComments)

	return &Lexer{scanner: s}
}
