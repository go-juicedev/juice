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

package juice

import "github.com/go-juicedev/juice/sql"

// Iterator is a type alias for sql.Iterator.
//
// This type is specifically designed for juicecli to use as a type annotation
// during code generation, allowing the tool to recognize and generate proper
// iterator-based query methods.
type Iterator[T any] = sql.Iterator[T]
