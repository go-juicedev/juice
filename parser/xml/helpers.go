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
	"strings"
)

func elementReadError(element string, err error) error {
	if err == io.EOF {
		return wrap(element, fmt.Errorf("element is not closed"))
	}
	var syntaxError *stdxml.SyntaxError
	if errors.As(err, &syntaxError) && strings.Contains(syntaxError.Msg, "unexpected EOF") {
		return wrap(element, fmt.Errorf("element is not closed: %w", err))
	}
	return err
}

func attributes(start stdxml.StartElement) map[string]string {
	if len(start.Attr) == 0 {
		return nil
	}
	attrs := make(map[string]string, len(start.Attr))
	for _, attr := range start.Attr {
		attrs[attr.Name.Local] = attr.Value
	}
	return attrs
}

func attribute(start stdxml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func requiredAttribute(start stdxml.StartElement, name string) (string, error) {
	value := attribute(start, name)
	if value == "" {
		return "", fmt.Errorf("attribute %q is required", name)
	}
	return value, nil
}

func parseText(decoder *stdxml.Decoder, end string) (string, error) {
	var builder strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return "", fmt.Errorf("element <%s> is not closed", end)
			}
			return "", err
		}
		switch token := token.(type) {
		case stdxml.CharData:
			builder.Write(token)
		case stdxml.StartElement:
			return "", fmt.Errorf("unexpected child element <%s>", token.Name.Local)
		case stdxml.EndElement:
			if token.Name.Local == end {
				return strings.TrimSpace(builder.String()), nil
			}
		}
	}
}

func skipElement(decoder *stdxml.Decoder, start stdxml.StartElement) error {
	if err := decoder.Skip(); err != nil {
		return wrap(start.Name.Local, err)
	}
	return nil
}
