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

package ctxreducer

import "context"

// ContextReducer is the interface that wraps the Reduce method.
// Accepts a context.Context and returns a new context instance that is the result of a transformation
// applied to the input context.
type ContextReducer interface {
	Reduce(ctx context.Context) context.Context
}

// ContextReducerFunc is an adapter to allow the use of ordinary functions as ContextReducer.
type ContextReducerFunc func(ctx context.Context) context.Context

// Reduce calls f(ctx).
func (f ContextReducerFunc) Reduce(ctx context.Context) context.Context {
	return f(ctx)
}

// ContextReducerGroup is a group of ContextReducer.
type ContextReducerGroup []ContextReducer

// Reduce calls each ContextReducer in the group.
func (g ContextReducerGroup) Reduce(ctx context.Context) context.Context {
	for _, r := range g {
		ctx = r.Reduce(ctx)
	}
	return ctx
}

type G = ContextReducerGroup
