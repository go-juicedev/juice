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

package parser

import (
	"io"
	"io/fs"
)

// Parser parses a configuration document into a format-independent model.
type Parser interface {
	Parse(io.Reader) (*Document, error)
}

// ParseFS opens name from fsys and parses it with p.
func ParseFS(fsys fs.FS, name string, p Parser) (*Document, error) {
	file, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return p.Parse(file)
}
