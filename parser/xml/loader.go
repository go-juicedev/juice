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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-juicedev/juice/parser"
)

var ErrUnexpectedHTTPStatus = errors.New("unexpected mapper HTTP status")

func (p *Parser) ParseFile(path string) (*parser.Document, error) {
	if p.FS == nil {
		return nil, errors.New("xml parser filesystem is required")
	}
	file, err := p.FS.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	document, entries, registry, err := p.parse(file)
	if err != nil {
		return nil, err
	}
	if err := p.loadMapperEntries(document, entries, registry); err != nil {
		return nil, err
	}
	return document, nil
}

func (p *Parser) loadMapperEntries(document *parser.Document, entries []mapperEntry, registry *sqlRegistry) error {
	if len(entries) == 0 {
		return nil
	}
	resolved := make([]parser.Mapper, 0, len(entries))
	for _, entry := range entries {
		if entry.mapper != nil {
			resolved = append(resolved, *entry.mapper)
			continue
		}
		if entry.source == nil {
			continue
		}
		source := *entry.source
		switch {
		case source.pattern != "":
			matches, err := fs.Glob(p.FS, source.pattern)
			if err != nil {
				return fmt.Errorf("invalid mapper pattern %q: %w", source.pattern, err)
			}
			for _, match := range matches {
				mapperDocument, err := p.loadMapperResource(match, registry)
				if err != nil {
					return err
				}
				resolved = append(resolved, mapperDocument)
			}
		case source.resource != "":
			mapperDocument, err := p.loadMapperResource(source.resource, registry)
			if err != nil {
				return err
			}
			resolved = append(resolved, mapperDocument)
		case source.url != "":
			mapperDocument, err := p.loadMapperURL(source.url, registry)
			if err != nil {
				return err
			}
			resolved = append(resolved, mapperDocument)
		}
	}
	document.Mappers = resolved
	return nil
}

func (p *Parser) loadMapperResource(resource string, registry *sqlRegistry) (parser.Mapper, error) {
	if p.FS == nil {
		return parser.Mapper{}, errors.New("xml parser filesystem is required")
	}
	file, err := p.FS.Open(resource)
	if err != nil {
		return parser.Mapper{}, err
	}
	defer func() { _ = file.Close() }()
	mapperDocument, err := parseMapperDocument(file, registry)
	if err != nil {
		return parser.Mapper{}, fmt.Errorf("failed to parse mapper %q: %w", resource, err)
	}
	return *mapperDocument, nil
}

func (p *Parser) loadMapperURL(rawURL string, registry *sqlRegistry) (parser.Mapper, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return parser.Mapper{}, err
	}
	switch parsedURL.Scheme {
	case "file":
		return p.loadMapperResource(strings.TrimPrefix(parsedURL.Path, "/"), registry)
	case "http", "https":
		return p.loadRemoteMapper(rawURL, registry)
	default:
		return parser.Mapper{}, fmt.Errorf("invalid mapper URL scheme %q", parsedURL.Scheme)
	}
}

func (p *Parser) loadRemoteMapper(rawURL string, registry *sqlRegistry) (parser.Mapper, error) {
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := client.Get(rawURL)
	if err != nil {
		return parser.Mapper{}, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return parser.Mapper{}, fmt.Errorf("%w: %s returned %s", ErrUnexpectedHTTPStatus, rawURL, response.Status)
	}
	mapperDocument, err := parseMapperDocument(response.Body, registry)
	if err != nil {
		return parser.Mapper{}, fmt.Errorf("failed to parse mapper %q: %w", rawURL, err)
	}
	return *mapperDocument, nil
}
