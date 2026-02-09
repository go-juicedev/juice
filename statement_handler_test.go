package juice

import (
	"context"
	stdsql "database/sql"
	sqldriver "database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	jdriver "github.com/go-juicedev/juice/driver"
	"github.com/go-juicedev/juice/eval"
	"github.com/go-juicedev/juice/session"
	jsql "github.com/go-juicedev/juice/sql"
)

type shSQLDriverState struct {
	prepareCalls   int
	stmtCloseCalls int
	stmtQueryCalls int
	stmtExecCalls  int
	connQueryCalls int
	connExecCalls  int
	beginCalls     int
	commitCalls    int
	rollbackCalls  int

	prepareErr  error
	queryErr    error
	execErr     error
	beginErr    error
	commitErr   error
	rollbackErr error
}

type shSQLDriver struct {
	state *shSQLDriverState
}

func (d *shSQLDriver) Open(_ string) (sqldriver.Conn, error) {
	return &shSQLConn{state: d.state}, nil
}

type shSQLConn struct {
	state *shSQLDriverState
}

func (c *shSQLConn) Prepare(query string) (sqldriver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *shSQLConn) PrepareContext(_ context.Context, _ string) (sqldriver.Stmt, error) {
	c.state.prepareCalls++
	if c.state.prepareErr != nil {
		return nil, c.state.prepareErr
	}
	return &shSQLStmt{state: c.state}, nil
}

func (c *shSQLConn) Close() error {
	return nil
}

func (c *shSQLConn) Begin() (sqldriver.Tx, error) {
	return c.BeginTx(context.Background(), sqldriver.TxOptions{})
}

func (c *shSQLConn) BeginTx(_ context.Context, _ sqldriver.TxOptions) (sqldriver.Tx, error) {
	c.state.beginCalls++
	if c.state.beginErr != nil {
		return nil, c.state.beginErr
	}
	return &shSQLTx{state: c.state}, nil
}

func (c *shSQLConn) ExecContext(_ context.Context, _ string, _ []sqldriver.NamedValue) (sqldriver.Result, error) {
	c.state.connExecCalls++
	if c.state.execErr != nil {
		return nil, c.state.execErr
	}
	return sqldriver.RowsAffected(1), nil
}

func (c *shSQLConn) QueryContext(_ context.Context, _ string, _ []sqldriver.NamedValue) (sqldriver.Rows, error) {
	c.state.connQueryCalls++
	if c.state.queryErr != nil {
		return nil, c.state.queryErr
	}
	return &shSQLRows{}, nil
}

var _ sqldriver.ConnPrepareContext = (*shSQLConn)(nil)
var _ sqldriver.ConnBeginTx = (*shSQLConn)(nil)
var _ sqldriver.ExecerContext = (*shSQLConn)(nil)
var _ sqldriver.QueryerContext = (*shSQLConn)(nil)

type shSQLStmt struct {
	state *shSQLDriverState
}

func (s *shSQLStmt) Close() error {
	s.state.stmtCloseCalls++
	return nil
}

func (s *shSQLStmt) NumInput() int {
	return -1
}

func (s *shSQLStmt) Exec(_ []sqldriver.Value) (sqldriver.Result, error) {
	return s.ExecContext(context.Background(), nil)
}

func (s *shSQLStmt) Query(_ []sqldriver.Value) (sqldriver.Rows, error) {
	return s.QueryContext(context.Background(), nil)
}

func (s *shSQLStmt) ExecContext(_ context.Context, _ []sqldriver.NamedValue) (sqldriver.Result, error) {
	s.state.stmtExecCalls++
	if s.state.execErr != nil {
		return nil, s.state.execErr
	}
	return sqldriver.RowsAffected(2), nil
}

func (s *shSQLStmt) QueryContext(_ context.Context, _ []sqldriver.NamedValue) (sqldriver.Rows, error) {
	s.state.stmtQueryCalls++
	if s.state.queryErr != nil {
		return nil, s.state.queryErr
	}
	return &shSQLRows{}, nil
}

var _ sqldriver.StmtExecContext = (*shSQLStmt)(nil)
var _ sqldriver.StmtQueryContext = (*shSQLStmt)(nil)

type shSQLRows struct{}

func (s *shSQLRows) Columns() []string {
	return []string{"value"}
}

func (s *shSQLRows) Close() error {
	return nil
}

func (s *shSQLRows) Next(_ []sqldriver.Value) error {
	return io.EOF
}

type shSQLTx struct {
	state *shSQLDriverState
}

func (t *shSQLTx) Commit() error {
	t.state.commitCalls++
	return t.state.commitErr
}

func (t *shSQLTx) Rollback() error {
	t.state.rollbackCalls++
	return t.state.rollbackErr
}

var shSQLDriverSeq uint64

func openStatementTestDB(t *testing.T, state *shSQLDriverState) *stdsql.DB {
	t.Helper()

	name := fmt.Sprintf("juice_statement_test_%d", atomic.AddUint64(&shSQLDriverSeq, 1))
	stdsql.Register(name, &shSQLDriver{state: state})

	db, err := stdsql.Open(name, "")
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

type shStatement struct {
	id      string
	name    string
	action  jsql.Action
	attrs   map[string]string
	buildFn func(translator jdriver.Translator, parameter eval.Parameter) (query string, args []any, err error)
}

func (s shStatement) ID() string {
	if s.id != "" {
		return s.id
	}
	return "id"
}

func (s shStatement) Name() string {
	if s.name != "" {
		return s.name
	}
	return "name"
}

func (s shStatement) Attribute(key string) string {
	return s.attrs[key]
}

func (s shStatement) Action() jsql.Action {
	if s.action != "" {
		return s.action
	}
	return jsql.Select
}

func (s shStatement) ResultMap() (jsql.ResultMap, error) {
	return nil, jsql.ErrResultMapNotSet
}

func (s shStatement) Build(translator jdriver.Translator, parameter eval.Parameter) (query string, args []any, err error) {
	if s.buildFn != nil {
		return s.buildFn(translator, parameter)
	}
	return "SELECT 1", nil, nil
}

type shNextStatementHandler struct {
	queryFn func(ctx context.Context, statement Statement, param eval.Param) (jsql.Rows, error)
	execFn  func(ctx context.Context, statement Statement, param eval.Param) (jsql.Result, error)
}

type shExecErrorMiddleware struct {
	err error
}

func (m shExecErrorMiddleware) QueryContext(_ Statement, _ Configuration, next QueryHandler) QueryHandler {
	return next
}

func (m shExecErrorMiddleware) ExecContext(_ Statement, _ Configuration, _ ExecHandler) ExecHandler {
	return func(_ context.Context, _ string, _ ...any) (jsql.Result, error) {
		return nil, m.err
	}
}

func (h shNextStatementHandler) QueryContext(ctx context.Context, statement Statement, param eval.Param) (jsql.Rows, error) {
	return h.queryFn(ctx, statement, param)
}

func (h shNextStatementHandler) ExecContext(ctx context.Context, statement Statement, param eval.Param) (jsql.Result, error) {
	return h.execFn(ctx, statement, param)
}

func TestBuildStatementQuery_statement_handler_test(t *testing.T) {
	stmt := shStatement{
		buildFn: func(translator jdriver.Translator, parameter eval.Parameter) (string, []any, error) {
			if got := translator.Translate("id"); got != "?" {
				t.Fatalf("unexpected translated placeholder: %q", got)
			}

			databaseID, ok := parameter.Get("_databaseId")
			if !ok || databaseID.String() != "sqlite3" {
				t.Fatalf("expected _databaseId sqlite3")
			}

			id, ok := parameter.Get("id")
			if !ok {
				t.Fatalf("expected id 7")
			}
			if got := reflect.ValueOf(id.Interface()); got.Kind() != reflect.Int || got.Int() != 7 {
				t.Fatalf("expected id 7")
			}

			nested, ok := parameter.Get("_parameter.id")
			if !ok {
				t.Fatalf("expected _parameter.id 7")
			}
			if got := reflect.ValueOf(nested.Interface()); got.Kind() != reflect.Int || got.Int() != 7 {
				t.Fatalf("expected _parameter.id 7")
			}

			return "SELECT ?", []any{id.Interface()}, nil
		},
	}

	query, args, err := buildStatementQuery(stmt, nil, &jdriver.SQLiteDriver{}, map[string]any{"id": 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if query != "SELECT ?" {
		t.Fatalf("unexpected query: %q", query)
	}

	if len(args) != 1 || args[0] != int64(7) && args[0] != 7 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestContextStatementHandler_statement_handler_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	param := H{"id": 1}
	stmt := shStatement{}

	next := shNextStatementHandler{
		queryFn: func(ctx context.Context, _ Statement, gotParam eval.Param) (jsql.Rows, error) {
			sess, err := session.FromContext(ctx)
			if err != nil || sess != db {
				t.Fatalf("expected db in session context, err=%v", err)
			}
			if !reflect.DeepEqual(eval.ParamFromContext(ctx), gotParam) {
				t.Fatalf("expected param in context")
			}
			return jsql.NewRowsBuffer([]string{"value"}, [][]any{}), nil
		},
		execFn: func(ctx context.Context, _ Statement, gotParam eval.Param) (jsql.Result, error) {
			sess, err := session.FromContext(ctx)
			if err != nil || sess != db {
				t.Fatalf("expected db in session context, err=%v", err)
			}
			if !reflect.DeepEqual(eval.ParamFromContext(ctx), gotParam) {
				t.Fatalf("expected param in context")
			}
			return sqldriver.RowsAffected(1), nil
		},
	}

	h := newContextStatementHandler(db, next)

	rows, err := h.QueryContext(context.Background(), stmt, param)
	if err != nil {
		t.Fatalf("unexpected query error: %v", err)
	}
	_ = rows.Close()

	result, err := h.ExecContext(context.Background(), stmt, param)
	if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected != 1 {
		t.Fatalf("unexpected rows affected: %d, err=%v", rowsAffected, err)
	}
}

func TestCompiledStatementHandler_statement_handler_test(t *testing.T) {
	stmt := shStatement{}
	h := newCompiledStatementHandler("SELECT ?", []any{1}, nil, &jdriver.SQLiteDriver{}, nil)

	qCalled := false
	_, err := h.WithQueryHandler(func(_ context.Context, query string, args ...any) (jsql.Rows, error) {
		qCalled = true
		if query != "SELECT ?" || len(args) != 1 || args[0] != 1 {
			t.Fatalf("unexpected query call: %s %#v", query, args)
		}
		return jsql.NewRowsBuffer([]string{"value"}, [][]any{}), nil
	}).QueryContext(context.Background(), stmt, nil)
	if err != nil {
		t.Fatalf("unexpected query error: %v", err)
	}
	if !qCalled {
		t.Fatalf("expected custom query handler called")
	}

	eCalled := false
	result, err := h.WithExecHandler(func(_ context.Context, query string, args ...any) (jsql.Result, error) {
		eCalled = true
		if query != "SELECT ?" || len(args) != 1 || args[0] != 1 {
			t.Fatalf("unexpected exec call: %s %#v", query, args)
		}
		return sqldriver.RowsAffected(2), nil
	}).ExecContext(context.Background(), stmt, nil)
	if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected != 2 {
		t.Fatalf("unexpected rows affected: %d, err=%v", rowsAffected, err)
	}
	if !eCalled {
		t.Fatalf("expected custom exec handler called")
	}

	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	ctx := session.WithContext(context.Background(), db)

	defaultHandler := newCompiledStatementHandler("SELECT 1", nil, nil, &jdriver.SQLiteDriver{}, nil)
	rows, err := defaultHandler.QueryContext(ctx, stmt, nil)
	if err != nil {
		t.Fatalf("unexpected default query error: %v", err)
	}
	if rows != nil {
		_ = rows.Close()
	}

	if _, err = defaultHandler.ExecContext(ctx, stmt, nil); err != nil {
		t.Fatalf("unexpected default exec error: %v", err)
	}

	if state.connQueryCalls == 0 || state.connExecCalls == 0 {
		t.Fatalf("expected default session handlers to hit db, query=%d exec=%d", state.connQueryCalls, state.connExecCalls)
	}
}

func TestPreparedStatementHandler_statement_handler_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	h := newPreparedStatementHandler(nil, &jdriver.SQLiteDriver{}, nil, db)
	ctx := context.Background()
	if err := newPreparedStatementHandler(nil, &jdriver.SQLiteDriver{}, nil, db).Close(); err != nil {
		t.Fatalf("unexpected empty close error: %v", err)
	}

	stmtQuery := shStatement{buildFn: func(_ jdriver.Translator, _ eval.Parameter) (string, []any, error) {
		return "SELECT 1", []any{1}, nil
	}}
	stmtExec := shStatement{buildFn: func(_ jdriver.Translator, _ eval.Parameter) (string, []any, error) {
		return "UPDATE t SET c = ?", []any{2}, nil
	}}

	rows, err := h.QueryContext(ctx, stmtQuery, nil)
	if err != nil {
		t.Fatalf("unexpected query error: %v", err)
	}
	if rows != nil {
		_ = rows.Close()
	}

	rows, err = h.QueryContext(ctx, stmtQuery, nil)
	if err != nil {
		t.Fatalf("unexpected second query error: %v", err)
	}
	if rows != nil {
		_ = rows.Close()
	}

	if _, err = h.ExecContext(ctx, stmtExec, nil); err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}

	if state.prepareCalls != 2 {
		t.Fatalf("expected 2 prepares, got %d", state.prepareCalls)
	}
	if state.stmtQueryCalls != 2 {
		t.Fatalf("expected 2 stmt queries, got %d", state.stmtQueryCalls)
	}
	if state.stmtExecCalls != 1 {
		t.Fatalf("expected 1 stmt exec, got %d", state.stmtExecCalls)
	}
	if state.stmtCloseCalls < 1 {
		t.Fatalf("expected stmt close called at least once, got %d", state.stmtCloseCalls)
	}

	if err = h.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	buildErr := errors.New("build failed")
	errStmt := shStatement{buildFn: func(_ jdriver.Translator, _ eval.Parameter) (string, []any, error) {
		return "", nil, buildErr
	}}

	if _, err = h.QueryContext(ctx, errStmt, nil); !errors.Is(err, buildErr) {
		t.Fatalf("expected build error, got %v", err)
	}

	if _, err = h.ExecContext(ctx, errStmt, nil); !errors.Is(err, buildErr) {
		t.Fatalf("expected build error, got %v", err)
	}

	state.prepareErr = errors.New("prepare failed")
	if _, err = h.QueryContext(ctx, stmtQuery, nil); err == nil || !strings.Contains(err.Error(), "prepare statement failed") {
		t.Fatalf("expected wrapped prepare error, got %v", err)
	}
	if _, err = h.ExecContext(ctx, stmtExec, nil); err == nil || !strings.Contains(err.Error(), "prepare statement failed") {
		t.Fatalf("expected wrapped prepare error in exec, got %v", err)
	}
}

func TestQueryBuildStatementHandler_statement_handler_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	h := newQueryBuildStatementHandler(&jdriver.SQLiteDriver{}, db, nil)
	ctx := context.Background()

	stmt := shStatement{buildFn: func(_ jdriver.Translator, _ eval.Parameter) (string, []any, error) {
		return "SELECT 1", nil, nil
	}}

	rows, err := h.QueryContext(ctx, stmt, nil)
	if err != nil {
		t.Fatalf("unexpected query error: %v", err)
	}
	if rows != nil {
		_ = rows.Close()
	}

	if _, err = h.ExecContext(ctx, stmt, nil); err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}

	if state.connQueryCalls == 0 || state.connExecCalls == 0 {
		t.Fatalf("expected db query and exec called")
	}

	buildErr := errors.New("build failed")
	errStmt := shStatement{buildFn: func(_ jdriver.Translator, _ eval.Parameter) (string, []any, error) {
		return "", nil, buildErr
	}}

	if _, err = h.QueryContext(ctx, errStmt, nil); !errors.Is(err, buildErr) {
		t.Fatalf("expected build error, got %v", err)
	}
	if _, err = h.ExecContext(ctx, errStmt, nil); !errors.Is(err, buildErr) {
		t.Fatalf("expected build error, got %v", err)
	}
}

func TestSliceMapAndBatchStatementHandlers_statement_handler_test(t *testing.T) {
	state := &shSQLDriverState{}
	db := openStatementTestDB(t, state)
	ctx := context.Background()

	stmt := shStatement{buildFn: func(_ jdriver.Translator, _ eval.Parameter) (string, []any, error) {
		return "INSERT INTO t(v) VALUES (?)", []any{1}, nil
	}}

	sliceHandler := newSliceBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, db, nil, reflect.ValueOf([]int{1}), 10)
	rows, err := sliceHandler.QueryContext(ctx, stmt, []int{1})
	if err != nil {
		t.Fatalf("unexpected slice query error: %v", err)
	}
	if rows != nil {
		_ = rows.Close()
	}

	if _, err = sliceHandler.execContext(ctx, stmt, []int{1}); err != nil {
		t.Fatalf("unexpected slice execContext error: %v", err)
	}

	if _, err = sliceHandler.ExecContext(ctx, stmt, []int{1}); err != nil {
		t.Fatalf("unexpected slice ExecContext error: %v", err)
	}

	multiSlice := []int{1, 2, 3}
	multiSliceHandler := newSliceBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, db, nil, reflect.ValueOf(multiSlice), 2)
	result, err := multiSliceHandler.ExecContext(ctx, stmt, multiSlice)
	if err != nil {
		t.Fatalf("unexpected multi-slice ExecContext error: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected != 4 {
		t.Fatalf("expected multi-slice rows affected 4, got %d err=%v", rowsAffected, err)
	}

	emptySliceHandler := newSliceBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, db, nil, reflect.ValueOf([]int{}), 10)
	if _, err = emptySliceHandler.ExecContext(ctx, stmt, []int{}); err == nil || !strings.Contains(err.Error(), "empty slice") {
		t.Fatalf("expected empty slice error, got %v", err)
	}

	mapHandler := newMapBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, nil, db, reflect.ValueOf(map[string][]int{"ids": {1}}), 10)
	rows, err = mapHandler.QueryContext(ctx, stmt, map[string][]int{"ids": {1}})
	if err != nil {
		t.Fatalf("unexpected map query error: %v", err)
	}
	if rows != nil {
		_ = rows.Close()
	}

	if _, err = mapHandler.execContext(ctx, stmt, map[string][]int{"ids": {1}}); err != nil {
		t.Fatalf("unexpected map execContext error: %v", err)
	}

	if _, err = mapHandler.ExecContext(ctx, stmt, map[string][]int{"ids": {1}}); err != nil {
		t.Fatalf("unexpected map ExecContext error: %v", err)
	}

	multiMap := map[string][]int{"ids": {1, 2, 3}}
	multiMapHandler := newMapBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, nil, db, reflect.ValueOf(multiMap), 2)
	result, err = multiMapHandler.ExecContext(ctx, stmt, multiMap)
	if err != nil {
		t.Fatalf("unexpected multi-map ExecContext error: %v", err)
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil || rowsAffected != 4 {
		t.Fatalf("expected multi-map rows affected 4, got %d err=%v", rowsAffected, err)
	}

	if _, err = newMapBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, nil, db, reflect.ValueOf(map[string][]int{"a": {1}, "b": {2}}), 10).ExecContext(ctx, stmt, nil); err == nil {
		t.Fatalf("expected map key count error")
	}

	if _, err = newMapBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, nil, db, reflect.ValueOf(map[int][]int{1: {1}}), 10).ExecContext(ctx, stmt, nil); err == nil {
		t.Fatalf("expected map key type error")
	}

	if _, err = newMapBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, nil, db, reflect.ValueOf(map[string]int{"ids": 1}), 10).ExecContext(ctx, stmt, nil); err == nil {
		t.Fatalf("expected map value type error")
	}

	if _, err = newMapBatchStatementHandler(&jdriver.SQLiteDriver{}, nil, nil, db, reflect.ValueOf(map[string][]int{"ids": {}}), 10).ExecContext(ctx, stmt, nil); err == nil {
		t.Fatalf("expected empty map slice error")
	}

	batchStmt := shStatement{
		attrs: map[string]string{"batchSize": "2"},
		buildFn: func(_ jdriver.Translator, _ eval.Parameter) (string, []any, error) {
			return "INSERT INTO t(v) VALUES (?)", []any{1}, nil
		},
	}

	batchHandler := newBatchStatementHandler(&jdriver.SQLiteDriver{}, db, nil)

	rows, err = batchHandler.QueryContext(ctx, stmt, nil)
	if err != nil {
		t.Fatalf("unexpected batch query error: %v", err)
	}
	if rows != nil {
		_ = rows.Close()
	}

	if _, err = batchHandler.execContext(ctx, stmt, []int{1}); err != nil {
		t.Fatalf("unexpected batch execContext error: %v", err)
	}

	if _, err = batchHandler.ExecContext(ctx, stmt, []int{1}); err != nil {
		t.Fatalf("unexpected batch exec without batchSize error: %v", err)
	}

	if _, err = batchHandler.ExecContext(ctx, shStatement{attrs: map[string]string{"batchSize": "bad"}, buildFn: stmt.buildFn}, []int{1}); err == nil {
		t.Fatalf("expected batch parse error")
	}

	if _, err = batchHandler.ExecContext(ctx, shStatement{attrs: map[string]string{"batchSize": "0"}, buildFn: stmt.buildFn}, []int{1}); err == nil {
		t.Fatalf("expected non-positive batch size error")
	}

	if _, err = batchHandler.ExecContext(ctx, batchStmt, 123); !errors.Is(err, errSliceOrArrayRequired) {
		t.Fatalf("expected errSliceOrArrayRequired, got %v", err)
	}

	if _, err = batchHandler.ExecContext(ctx, batchStmt, []int{1}); err != nil {
		t.Fatalf("unexpected batch slice exec error: %v", err)
	}

	if _, err = batchHandler.ExecContext(ctx, batchStmt, map[string][]int{"ids": {1}}); err != nil {
		t.Fatalf("unexpected batch map exec error: %v", err)
	}

	skipErr := fmt.Errorf("skip this batch: %w", ErrBatchSkip)
	skipSliceHandler := newSliceBatchStatementHandler(&jdriver.SQLiteDriver{}, MiddlewareGroup{shExecErrorMiddleware{err: skipErr}}, db, nil, reflect.ValueOf(multiSlice), 2)
	if _, err = skipSliceHandler.ExecContext(ctx, stmt, multiSlice); err == nil || !errors.Is(err, ErrBatchSkip) {
		t.Fatalf("expected ErrBatchSkip from slice batch, got %v", err)
	}

	skipMapHandler := newMapBatchStatementHandler(&jdriver.SQLiteDriver{}, MiddlewareGroup{shExecErrorMiddleware{err: skipErr}}, nil, db, reflect.ValueOf(multiMap), 2)
	if _, err = skipMapHandler.ExecContext(ctx, stmt, multiMap); err == nil || !errors.Is(err, ErrBatchSkip) {
		t.Fatalf("expected ErrBatchSkip from map batch, got %v", err)
	}

	nonSkipErr := errors.New("hard failure")
	nonSkipSliceHandler := newSliceBatchStatementHandler(&jdriver.SQLiteDriver{}, MiddlewareGroup{shExecErrorMiddleware{err: nonSkipErr}}, db, nil, reflect.ValueOf(multiSlice), 2)
	if _, err = nonSkipSliceHandler.ExecContext(ctx, stmt, multiSlice); !errors.Is(err, nonSkipErr) {
		t.Fatalf("expected non-skip error from slice batch, got %v", err)
	}

	nonSkipMapHandler := newMapBatchStatementHandler(&jdriver.SQLiteDriver{}, MiddlewareGroup{shExecErrorMiddleware{err: nonSkipErr}}, nil, db, reflect.ValueOf(multiMap), 2)
	if _, err = nonSkipMapHandler.ExecContext(ctx, stmt, multiMap); !errors.Is(err, nonSkipErr) {
		t.Fatalf("expected non-skip error from map batch, got %v", err)
	}
}
