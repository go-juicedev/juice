package stmt

import (
	"context"
	"database/sql"
	"testing"
	"unsafe"
)

func createTestStmt(query string) *sql.Stmt {
	s := stmt{query: query}
	return (*sql.Stmt)(unsafe.Pointer(&s))
}

func TestFromContext(t *testing.T) {
	tests := []struct {
		name      string
		ctxQuery  string
		findQuery string
		want      bool
	}{
		{
			name:      "found query",
			ctxQuery:  "SELECT * FROM users WHERE id = ?",
			findQuery: "SELECT * FROM users WHERE id = ?",
			want:      true,
		},
		{
			name:      "query not match",
			ctxQuery:  "SELECT * FROM users WHERE id = ?",
			findQuery: "SELECT * FROM posts WHERE id = ?",
			want:      false,
		},
		{
			name:      "empty query",
			ctxQuery:  "",
			findQuery: "SELECT * FROM users",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStmt := createTestStmt(tt.ctxQuery)

			ctx := context.Background()
			ctx = WithContext(ctx, sqlStmt)

			got, ok := FromContext(ctx, tt.findQuery)
			if ok != tt.want {
				t.Errorf("FromContext() ok = %v, want %v", ok, tt.want)
				return
			}

			if ok && Query(got) != tt.ctxQuery {
				t.Errorf("FromContext() query = %v, want %v", Query(got), tt.ctxQuery)
			}
		})
	}
}

func TestFromContextChain(t *testing.T) {
	stmt1 := createTestStmt("SELECT * FROM users")
	stmt2 := createTestStmt("SELECT * FROM posts")
	stmt3 := createTestStmt("SELECT * FROM comments")

	ctx := context.Background()
	ctx = WithContext(ctx, stmt1)
	ctx = WithContext(ctx, stmt2)
	ctx = WithContext(ctx, stmt3)

	if got, ok := FromContext(ctx, "SELECT * FROM users"); !ok {
		t.Error("FromContext() should find first statement")
	} else if Query(got) != "SELECT * FROM users" {
		t.Errorf("FromContext() = %v, want %v", Query(got), "SELECT * FROM users")
	}
}

func TestFromContextCircular(t *testing.T) {
	stmt := createTestStmt("SELECT * FROM users")

	ctx := context.Background()
	ctx = WithContext(ctx, stmt)
	ctx = WithContext(ctx, stmt)

	if _, ok := FromContext(ctx, "SELECT * FROM posts"); ok {
		t.Error("FromContext() should not find non-existent query")
	}
}

func BenchmarkFromContext(b *testing.B) {
	stmt := createTestStmt("SELECT * FROM users WHERE id = ?")
	ctx := WithContext(context.Background(), stmt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromContext(ctx, "SELECT * FROM users WHERE id = ?")
	}
}

func BenchmarkFromContextChain(b *testing.B) {
	stmt1 := createTestStmt("SELECT * FROM users")
	stmt2 := createTestStmt("SELECT * FROM posts")
	stmt3 := createTestStmt("SELECT * FROM comments")

	ctx := context.Background()
	ctx = WithContext(ctx, stmt1)
	ctx = WithContext(ctx, stmt2)
	ctx = WithContext(ctx, stmt3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FromContext(ctx, "SELECT * FROM users")
	}
}
