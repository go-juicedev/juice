package session_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/go-juicedev/juice/session"
)

type dummySession struct{}

func (dummySession) QueryContext(context.Context, string, ...any) (*sql.Rows, error) { return nil, nil }
func (dummySession) ExecContext(context.Context, string, ...any) (sql.Result, error) { return nil, nil }
func (dummySession) PrepareContext(context.Context, string) (*sql.Stmt, error)        { return nil, nil }

func TestFromContext_NoSession_context_test(t *testing.T) {
	_, err := session.FromContext(context.Background())
	if !errors.Is(err, session.ErrNoSession) {
		t.Fatalf("unexpected error: got=%v want=%v", err, session.ErrNoSession)
	}
}

func TestWithContext_RoundTrip_context_test(t *testing.T) {
	want := dummySession{}
	ctx := session.WithContext(context.Background(), want)

	got, err := session.FromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected session: got=%T want=%T", got, want)
	}
}

func TestWithContext_NilSession_context_test(t *testing.T) {
	ctx := session.WithContext(context.Background(), session.Session(nil))
	_, err := session.FromContext(ctx)
	if !errors.Is(err, session.ErrNoSession) {
		t.Fatalf("unexpected error: got=%v want=%v", err, session.ErrNoSession)
	}
}

