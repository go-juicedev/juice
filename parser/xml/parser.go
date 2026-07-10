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

import (
	stdxml "encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"

	"github.com/go-juicedev/juice/parser"
)

var ErrMapperRootElementNotFound = errors.New("mapper root element <mapper> not found")

type Parser struct {
	FS                fs.FS
	Client            *http.Client
	IgnoreEnvironment bool
}

var _ parser.Parser = (*Parser)(nil)

func (p *Parser) Parse(reader io.Reader) (*parser.Document, error) {
	decoder := stdxml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("configuration root element not found")
			}
			return nil, err
		}
		start, ok := token.(stdxml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "configuration" {
			return nil, wrap(start.Name.Local, fmt.Errorf("expected <configuration> root element"))
		}
		return p.parseConfiguration(decoder)
	}
}

func Parse(reader io.Reader) (*parser.Document, error) {
	return new(Parser).Parse(reader)
}

func ParseMapper(reader io.Reader) (*parser.Mapper, error) {
	decoder := stdxml.NewDecoder(reader)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil, ErrMapperRootElementNotFound
			}
			return nil, err
		}
		start, ok := token.(stdxml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "mapper" {
			return nil, wrap(start.Name.Local, ErrMapperRootElementNotFound)
		}
		mapperDocument, err := parseMapper(decoder, start)
		if err != nil {
			return nil, err
		}
		return &mapperDocument, nil
	}
}
