/*
Copyright 2024 eatmoreapple

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

package stmt

import (
	"context"
	"database/sql"
)

// stmtCtx is the key type for context values.
// Using empty struct to minimize memory usage.
type stmtCtx struct{}

// stmtValue holds a prepared statement and its parent context,
// forming a chain-like structure for statement lookup.
type stmtValue struct {
	ctx  context.Context
	stmt *sql.Stmt
}

// FromContext retrieves a SQL statement from the context chain that matches the given query.
// It performs a recursive search through the context chain until it finds a matching statement
// or reaches the end of the chain.
//
// Note: It prevents infinite recursion by checking for self-referential contexts.
func FromContext(ctx context.Context, query string) (*sql.Stmt, bool) {
	value, ok := ctx.Value(stmtCtx{}).(*stmtValue)
	if !ok {
		return nil, false
	}
	if Query(value.stmt) != query {
		if value.ctx == ctx { // for circular references check
			return nil, false
		}
		return FromContext(value.ctx, query)
	}
	return value.stmt, true
}

// WithContext creates a new context containing the provided SQL statement.
// It maintains a chain of contexts, allowing for statement reuse and lookup.
func WithContext(ctx context.Context, stmt *sql.Stmt) context.Context {
	return context.WithValue(ctx, stmtCtx{}, &stmtValue{ctx: ctx, stmt: stmt})
}
