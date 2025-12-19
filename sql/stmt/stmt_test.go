package stmt

import (
	"database/sql"
	"testing"
	"unsafe"
)

type stmt struct {
	db    *sql.DB
	query string
}

func TestQuery(t *testing.T) {
	const query = "select * from user where id = ?"
	var s = stmt{query: query}

	// unsafe case to sql.Stmt
	sqlStmt := (*sql.Stmt)(unsafe.Pointer(&s))

	if Query(sqlStmt) != query {
		t.Errorf("Query() = %q; want %q", Query(sqlStmt), query)
	}
}

func BenchmarkQuery(b *testing.B) {
	const query = "select * from user where id = ?"
	var s = stmt{query: query}

	// unsafe case to sql.Stmt
	sqlStmt := (*sql.Stmt)(unsafe.Pointer(&s))

	for i := 0; i < b.N; i++ {
		if Query(sqlStmt) != query {
			b.Errorf("Query() = %q; want %q", Query(sqlStmt), query)
		}
	}
}
