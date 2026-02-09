package sql

import (
	"database/sql/driver"
	"errors"
	"testing"
)

func TestAction_Methods_action_result_test(t *testing.T) {
	if Select.String() != "select" {
		t.Fatalf("unexpected select string: %q", Select.String())
	}

	if !Select.ForRead() {
		t.Fatalf("select should be read action")
	}

	if Select.ForWrite() {
		t.Fatalf("select should not be write action")
	}

	for _, action := range []Action{Insert, Update, Delete} {
		if action.ForRead() {
			t.Fatalf("%s should not be read action", action)
		}
		if !action.ForWrite() {
			t.Fatalf("%s should be write action", action)
		}
	}
}

type resultStub struct {
	lastInsertID int64
	rowsAffected int64
	lastInsertErr error
	rowsErr       error
}

func (r resultStub) LastInsertId() (int64, error) {
	if r.lastInsertErr != nil {
		return 0, r.lastInsertErr
	}
	return r.lastInsertID, nil
}

func (r resultStub) RowsAffected() (int64, error) {
	if r.rowsErr != nil {
		return 0, r.rowsErr
	}
	return r.rowsAffected, nil
}

var _ driver.Result = resultStub{}

func TestBatchResult_AccumulateResult_action_result_test(t *testing.T) {
	b := &BatchResult{}

	b.AccumulateResult(nil)

	b.AccumulateResult(resultStub{lastInsertID: 1, rowsAffected: 2})
	b.AccumulateResult(resultStub{lastInsertID: 5, rowsAffected: 3})
	b.AccumulateResult(resultStub{lastInsertErr: errors.New("id err"), rowsAffected: 4})
	b.AccumulateResult(resultStub{lastInsertID: 9, rowsErr: errors.New("rows err")})

	rows, err := b.RowsAffected()
	if err != nil {
		t.Fatalf("unexpected rows error: %v", err)
	}
	if rows != 9 {
		t.Fatalf("expected total rows 9, got %d", rows)
	}

	id, err := b.LastInsertId()
	if err != nil {
		t.Fatalf("unexpected id error: %v", err)
	}
	if id != 9 {
		t.Fatalf("expected last id 9, got %d", id)
	}
}
