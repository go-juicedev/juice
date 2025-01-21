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

import (
	"context"

	"github.com/go-juicedev/juice/session"
)

// SessionWithContextReducer is a ContextReducer that adds a Session to the context.
type sessionWithContextReducer struct {
	session session.Session
}

// The Reduce method uses an external function SessionWithContext to add the Session to the context.
func (r sessionWithContextReducer) Reduce(ctx context.Context) context.Context {
	return session.WithContext(ctx, r.session)
}

// NewSessionContextReducer returns a new instance of the sessionWithContextReducer.
func NewSessionContextReducer(session session.Session) ContextReducer {
	return sessionWithContextReducer{session: session}
}
