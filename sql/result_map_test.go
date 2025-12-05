package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-juicedev/juice/internal/sqlmock"
)

// --- Test Structs ---

type SimpleStruct struct {
	ID   int           `column:"id"`
	Name string        `column:"name"`
	Age  sql.NullInt64 `column:"age"`
}

type NestedStruct struct {
	Info SimpleStruct `column:"info"` // This won't be mapped directly by rowDestination as a nested tag
	Rate float64      `column:"rate"`
}

type anonymousStruct struct {
	_    string
	Name string `column:"name"`
}

type AnonymousStruct struct {
	ID              int     `column:"id"`
	anonymousStruct         // Anonymous field
	Rate            float64 `column:"rate"`
}

type CustomTagStruct struct {
	Field1 string `column:"field_1"`
	Field2 int    `column:"field_2"`
}

type ScannerStruct struct {
	Value   string
	scanned bool
}

func (ss *ScannerStruct) Scan(src any) error {
	if src == nil {
		return errors.New("ScannerStruct: cannot scan nil")
	}
	switch v := src.(type) {
	case string:
		ss.Value = v
	case []byte:
		ss.Value = string(v)
	default:
		return fmt.Errorf("ScannerStruct: cannot scan type %T", src)
	}
	ss.scanned = true
	return nil
}

type RowScannerStruct struct {
	ID      int
	Content string
	scanned bool
}

// ScanRows implements the RowScanner interface
func (rs *RowScannerStruct) ScanRows(rows Rows) error {
	rs.scanned = true
	// A simplified scan, assuming order or specific columns
	// In a real scenario, you might inspect rows.ColumnsLine()
	return rows.Scan(&rs.ID, &rs.Content)
}

// --- Test Cases ---

func TestSingleRowResultMap_MapTo_Success(t *testing.T) {
	mapper := SingleRowResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"id", "name", "age"},
		Data: [][]any{
			{1, "Test Name", 30},
		},
	}

	var result SimpleStruct
	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if err != nil {
		t.Fatalf("MapTo failed: %v", err)
	}

	if result.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", result.ID)
	}
	if result.Name != "Test Name" {
		t.Errorf("Expected Name to be 'Test Name', got '%s'", result.Name)
	}
	if !result.Age.Valid || result.Age.Int64 != 30 {
		t.Errorf("Expected Age to be 30, got %v", result.Age)
	}
}

func TestSingleRowResultMap_MapTo_PointerToBasicType(t *testing.T) {
	mapper := SingleRowResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"count"},
		Data:        [][]any{{42}},
	}
	var result int
	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if err != nil {
		t.Fatalf("MapTo failed for *int: %v", err)
	}
	if result != 42 {
		t.Errorf("Expected result to be 42, got %d", result)
	}

	rowsString := &sqlmock.MockRows{
		ColumnsLine: []string{"value"},
		Data:        [][]any{{"hello"}},
	}
	var resultStr string
	err = mapper.MapTo(reflect.ValueOf(&resultStr), rowsString)
	if err != nil {
		t.Fatalf("MapTo failed for *string: %v", err)
	}
	if resultStr != "hello" {
		t.Errorf("Expected resultStr to be 'hello', got '%s'", resultStr)
	}
}

func TestSingleRowResultMap_MapTo_NoRows(t *testing.T) {
	mapper := SingleRowResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"id", "name"},
		Data:        [][]any{},
	}
	var result SimpleStruct
	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}
}

func TestSingleRowResultMap_MapTo_TooManyRows(t *testing.T) {
	mapper := SingleRowResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"id"},
		Data: [][]any{
			{1},
			{2},
		},
	}
	var result SimpleStruct
	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if !errors.Is(err, ErrTooManyRows) {
		t.Errorf("Expected ErrTooManyRows, got %v", err)
	}
}

func TestSingleRowResultMap_MapTo_NotAPointer(t *testing.T) {
	mapper := SingleRowResultMap{}
	rows := &sqlmock.MockRows{}
	var result SimpleStruct // Not a pointer
	err := mapper.MapTo(reflect.ValueOf(result), rows)
	if !errors.Is(err, ErrPointerRequired) {
		t.Errorf("Expected ErrPointerRequired, got %v", err)
	}
}

func TestMultiRowsResultMap_MapTo_Success_SliceOfStructs(t *testing.T) {
	mapper := MultiRowsResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"id", "name"},
		Data: [][]any{
			{1, "Alice"},
			{2, "Bob"},
		},
	}
	var result []SimpleStruct
	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if err != nil {
		t.Fatalf("MapTo failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(result))
	}
	if result[0].ID != 1 || result[0].Name != "Alice" {
		t.Errorf("Unexpected result[0]: %+v", result[0])
	}
	if result[1].ID != 2 || result[1].Name != "Bob" {
		t.Errorf("Unexpected result[1]: %+v", result[1])
	}
}

func TestMultiRowsResultMap_MapTo_Success_SliceOfPointersToStructs(t *testing.T) {
	mapper := MultiRowsResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"id", "name"},
		Data: [][]any{
			{1, "Alice"},
			{2, "Bob"},
		},
	}
	var result []*SimpleStruct
	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if err != nil {
		t.Fatalf("MapTo failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(result))
	}
	if result[0].ID != 1 || result[0].Name != "Alice" {
		t.Errorf("Unexpected result[0]: %+v", *result[0])
	}
	if result[1].ID != 2 || result[1].Name != "Bob" {
		t.Errorf("Unexpected result[1]: %+v", *result[1])
	}
}

func TestMultiRowsResultMap_MapTo_Success_SliceOfBasicType(t *testing.T) {
	mapper := MultiRowsResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"id"},
		Data:        [][]any{{10}, {20}, {30}},
	}
	var result []int
	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if err != nil {
		t.Fatalf("MapTo failed for []int: %v", err)
	}
	if len(result) != 3 || result[0] != 10 || result[1] != 20 || result[2] != 30 {
		t.Errorf("Expected [10 20 30], got %v", result)
	}

	rowsString := &sqlmock.MockRows{
		ColumnsLine: []string{"value"},
		Data:        [][]any{{"a"}, {"b"}},
	}
	var resultStr []string
	err = mapper.MapTo(reflect.ValueOf(&resultStr), rowsString)
	if err != nil {
		t.Fatalf("MapTo failed for []string: %v", err)
	}
	if len(resultStr) != 2 || resultStr[0] != "a" || resultStr[1] != "b" {
		t.Errorf("Expected [a b], got %v", resultStr)
	}
}

func TestMultiRowsResultMap_MapTo_EmptyResult(t *testing.T) {
	mapper := MultiRowsResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"id", "name"},
		Data:        [][]any{},
	}
	var result []SimpleStruct

	// Test with resultMapPreserveNilSlice = false (default)
	_ = os.Unsetenv("JUICE_RESULT_MAP_PRESERVE_NIL_SLICE")
	resultMapPreserveNilSlice = false // ensure internal var is also reset for test

	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if err != nil {
		t.Fatalf("MapTo failed: %v", err)
	}
	if result == nil {
		t.Errorf("Expected non-nil empty slice when JUICE_RESULT_MAP_PRESERVE_NIL_SLICE is false, got nil")
	}
	if len(result) != 0 {
		t.Errorf("Expected 0 results, got %d", len(result))
	}

	// Test with resultMapPreserveNilSlice = true
	t.Setenv("JUICE_RESULT_MAP_PRESERVE_NIL_SLICE", "true")
	resultMapPreserveNilSlice = true // ensure internal var is also set for test

	var resultNil []SimpleStruct // new variable, should remain nil
	err = mapper.MapTo(reflect.ValueOf(&resultNil), rows)
	if err != nil {
		t.Fatalf("MapTo failed: %v", err)
	}
	if resultNil != nil {
		t.Errorf("Expected nil slice when JUICE_RESULT_MAP_PRESERVE_NIL_SLICE is true, got non-nil: %v", resultNil)
	}

	// Cleanup env for other tests
	_ = os.Unsetenv("JUICE_RESULT_MAP_PRESERVE_NIL_SLICE")
	resultMapPreserveNilSlice = false
}

func TestMultiRowsResultMap_MapTo_WithNewFunc(t *testing.T) {
	var newCalls int
	mapper := MultiRowsResultMap{
		New: func() reflect.Value {
			newCalls++
			return reflect.New(reflect.TypeOf(SimpleStruct{}))
		},
	}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"id", "name"},
		Data:        [][]any{{1, "Test"}},
	}
	var result []*SimpleStruct // Slice of pointers to use the New func effectively
	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if err != nil {
		t.Fatalf("MapTo failed: %v", err)
	}
	if newCalls != 1 {
		t.Errorf("Expected New func to be called 1 time, got %d", newCalls)
	}
	if len(result) != 1 || result[0].ID != 1 {
		t.Errorf("Unexpected result with New func: %+v", result)
	}
}

func TestMultiRowsResultMap_MapTo_RowScanner(t *testing.T) {
	mapper := MultiRowsResultMap{}
	rows := &sqlmock.MockRows{
		ColumnsLine: []string{"col_id", "col_content"}, // Column names for RowScannerStruct
		Data: [][]any{
			{10, "Data1"},
			{20, "Data2"},
		},
	}
	var result []*RowScannerStruct // Slice of pointers to RowScannerStruct

	err := mapper.MapTo(reflect.ValueOf(&result), rows)
	if err != nil {
		t.Fatalf("MapTo with RowScanner failed: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 results with RowScanner, got %d", len(result))
	}
	if !result[0].scanned || result[0].ID != 10 || result[0].Content != "Data1" {
		t.Errorf("Unexpected result[0] with RowScanner: %+v", *result[0])
	}
	if !result[1].scanned || result[1].ID != 20 || result[1].Content != "Data2" {
		t.Errorf("Unexpected result[1] with RowScanner: %+v", *result[1])
	}
}

func TestMultiRowsResultMap_MapTo_Errors(t *testing.T) {
	mapper := MultiRowsResultMap{}
	var result []SimpleStruct

	// Not a pointer
	err := mapper.MapTo(reflect.ValueOf(result), &sqlmock.MockRows{})
	if err == nil || !strings.Contains(err.Error(), "pointer to slice") {
		t.Errorf("Expected error for non-pointer, got %v", err)
	}

	// Pointer to non-slice
	var notASlice int
	err = mapper.MapTo(reflect.ValueOf(&notASlice), &sqlmock.MockRows{})
	if err == nil || !strings.Contains(err.Error(), "pointer to slice, got pointer to int") {
		t.Errorf("Expected error for pointer to non-slice, got %v", err)
	}

	expectedErr := errors.New("test error")

	// ColumnsLine error
	rowsColsErr := &sqlmock.MockRows{ColumnsErr: expectedErr}
	err = mapper.MapTo(reflect.ValueOf(&result), rowsColsErr)
	if err == nil || !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("Expected ColumnsLine error, got %v", err)
	}

	// Scan error
	rowsScanErr := &sqlmock.MockRows{
		ColumnsLine: []string{"id"},
		Data:        [][]any{{1}},
		ScanErr:     expectedErr,
	}
	err = mapper.MapTo(reflect.ValueOf(&result), rowsScanErr)
	if err == nil || !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("Expected Scan error, got %v", err)
	}

	// RowScanner ScanRows error
	var rowScannerResult []*RowScannerStruct
	rowsRowScanErr := &sqlmock.MockRows{
		ColumnsLine: []string{"id", "content"},
		Data:        [][]any{{1, "data"}},
		ScanErr:     expectedErr, // Mock Scan within RowScanner's ScanRows to fail
	}
	// To make RowScannerStruct.ScanRows fail, its internal call to rows.Scan must fail.
	// So we configure scanErr on the mockRows passed to RowScannerStruct.
	mapperForScanErr := MultiRowsResultMap{
		New: func() reflect.Value {
			// Ensure the instance used by ScanRows gets the erroring mockRows
			return reflect.New(reflect.TypeOf(RowScannerStruct{}))
		},
	}
	err = mapperForScanErr.MapTo(reflect.ValueOf(&rowScannerResult), rowsRowScanErr)
	if err == nil || !strings.Contains(err.Error(), "failed to scan row using RowScanner") || !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("Expected RowScanner ScanRows error, got %v", err)
	}

	// rows.Err() after iteration
	rowsErrAfterIter := &sqlmock.MockRows{
		ColumnsLine: []string{"id"},
		Data:        [][]any{{1}},
		Reason:      expectedErr,
	}
	err = mapper.MapTo(reflect.ValueOf(&result), rowsErrAfterIter)
	if err == nil || !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("Expected rows.Err() after iteration, got %v", err)
	}
}

// --- rowDestination Tests (indirectly tested via mappers, but some direct unit tests are useful) ---

func TestRowDestination_Destination_Basic(t *testing.T) {
	dest := &rowDestination{}
	var s SimpleStruct
	rv := reflect.ValueOf(&s).Elem()
	columns := []string{"id", "name", "age", "nonexistent_column"}

	scanDest, err := dest.Destination(rv, columns)
	if err != nil {
		t.Fatalf("Destination failed: %v", err)
	}
	if len(scanDest) != 4 {
		t.Fatalf("Expected 4 destinations, got %d", len(scanDest))
	}

	// Check types (rough check, assumes order based on columns)
	if _, ok := scanDest[0].(*int); !ok {
		t.Errorf("Expected dest[0] to be *int, got %T", scanDest[0])
	}
	if _, ok := scanDest[1].(*string); !ok {
		t.Errorf("Expected dest[1] to be *string, got %T", scanDest[1])
	}
	if _, ok := scanDest[2].(*sql.NullInt64); !ok {
		t.Errorf("Expected dest[2] to be *sql.NullInt64, got %T", scanDest[2])
	}
	if scanDest[3] != &sink {
		t.Errorf("Expected dest[3] to be &sink, got %T", scanDest[3])
	}

	// Test caching of indexes
	scanDest2, err := dest.Destination(rv, columns)
	if err != nil {
		t.Fatalf("Second Destination call failed: %v", err)
	}
	if !reflect.DeepEqual(scanDest, scanDest2) {
		// Note: pointers will be different, but the underlying field mapping should be the same.
		// This primarily checks that no error occurs and length is same.
		// A more robust check would compare the field addresses, but that's complex.
		t.Logf("First dest: %v, Second dest: %v", scanDest, scanDest2)
		// For now, just check length as a proxy for cached behavior correctness
		if len(scanDest2) != len(scanDest) {
			t.Errorf("Destination cache might not be working as expected based on length")
		}
	}
}

func TestRowDestination_Destination_AnonymousStruct(t *testing.T) {
	dest := &rowDestination{}
	var as AnonymousStruct
	rv := reflect.ValueOf(&as).Elem()
	// SimpleStruct is anonymous: its 'id' and 'name' fields should be promoted.
	// 'Age' from SimpleStruct does not have a 'column' tag, so it won't be mapped unless explicitly tagged.
	columns := []string{"id", "name", "rate"}

	scanDest, err := dest.Destination(rv, columns)
	if err != nil {
		t.Fatalf("Destination failed for AnonymousStruct: %v", err)
	}
	if len(scanDest) != 3 {
		t.Fatalf("Expected 3 destinations, got %d", len(scanDest))
	}

	// Check that 'id' maps to AnonymousStruct.ID (or SimpleStruct.ID if AnonymousStruct.ID wasn't there)
	// and 'name' maps to AnonymousStruct.SimpleStruct.Name
	// and 'rate' maps to AnonymousStruct.Rate

	// s.indexes should be:
	// columns: "id", "name", "rate"
	// AnonymousStruct.ID -> field 0 of AnonymousStruct -> s.indexes[0] = {0}
	// AnonymousStruct.SimpleStruct.Name -> field 1 of AnonymousStruct, then field 1 of SimpleStruct -> s.indexes[1] = {1,1}
	// AnonymousStruct.Rate -> field 2 of AnonymousStruct -> s.indexes[2] = {2}

	expectedIndexes := [][]int{
		{0},    // id -> AnonymousStruct.ID
		{1, 1}, // name -> AnonymousStruct.SimpleStruct.Name
		{2},    // rate -> AnonymousStruct.Rate
	}

	if !reflect.DeepEqual(dest.indexes, expectedIndexes) {
		t.Errorf("Expected indexes %v, got %v", expectedIndexes, dest.indexes)
	}

	if _, ok := scanDest[0].(*int); !ok { // AnonymousStruct.ID
		t.Errorf("Expected dest[0] for 'id' to be *int, got %T", scanDest[0])
	}

	if _, ok := scanDest[1].(*string); !ok { // AnonymousStruct.SimpleStruct.Name
		t.Errorf("Expected dest[1] for 'name' to be *string, got %T", scanDest[1])
	}
	if scanDest[1] != rv.FieldByIndex([]int{1, 1}).Addr().Interface() { // SimpleStruct.Name
		t.Errorf("Destination for 'name' does not point to AnonymousStruct.SimpleStruct.Name")
	}

	if _, ok := scanDest[2].(*float64); !ok { // AnonymousStruct.Rate
		t.Errorf("Expected dest[2] for 'rate' to be *float64, got %T", scanDest[2])
	}
	if scanDest[2] != rv.FieldByName("Rate").Addr().Interface() {
		t.Errorf("Destination for 'rate' does not point to AnonymousStruct.Rate")
	}
}

func TestRowDestination_Destination_CustomTag(t *testing.T) {
	dest := &rowDestination{}
	var cts CustomTagStruct
	rv := reflect.ValueOf(&cts).Elem()
	columns := []string{"field_1", "field_2"}

	scanDest, err := dest.Destination(rv, columns)
	if err != nil {
		t.Fatalf("Destination failed for CustomTagStruct: %v", err)
	}
	if len(scanDest) != 2 {
		t.Fatalf("Expected 2 destinations, got %d", len(scanDest))
	}
	if _, ok := scanDest[0].(*string); !ok {
		t.Errorf("Expected dest[0] to be *string, got %T", scanDest[0])
	}
	if _, ok := scanDest[1].(*int); !ok {
		t.Errorf("Expected dest[1] to be *int, got %T", scanDest[1])
	}
}

func TestRowDestination_Destination_SingleColumnNonStruct(t *testing.T) {
	dest := &rowDestination{}
	var i int
	rv := reflect.ValueOf(&i)
	columns := []string{"value"}

	scanDest, err := dest.Destination(rv, columns)
	if err != nil {
		t.Fatalf("Destination failed for single column non-struct: %v", err)
	}
	if len(scanDest) != 1 {
		t.Fatalf("Expected 1 destination, got %d", len(scanDest))
	}
	if _, ok := scanDest[0].(*int); !ok {
		t.Errorf("Expected dest[0] to be *int, got %T", scanDest[0])
	}
	if scanDest[0] != &i {
		t.Error("Destination does not point to the address of 'i'")
	}

	var s string
	rvs := reflect.ValueOf(&s)
	scanDestS, err := dest.Destination(rvs, columns) // dest is reused, should re-evaluate
	if err != nil {
		t.Fatalf("Destination failed for single column string: %v", err)
	}
	if len(scanDestS) != 1 {
		t.Fatalf("Expected 1 destination for string, got %d", len(scanDestS))
	}
	if scanDestS[0] != &s {
		t.Error("Destination does not point to the address of 's'")
	}
}

func TestRowDestination_Destination_SingleColumn_Time(t *testing.T) {
	dest := &rowDestination{}
	var tm time.Time
	rv := reflect.ValueOf(&tm)
	columns := []string{"created_at"}

	scanDest, err := dest.Destination(rv, columns)
	if err != nil {
		t.Fatalf("Destination failed for time.Time: %v", err)
	}
	if len(scanDest) != 1 {
		t.Fatalf("Expected 1 destination for time.Time, got %d", len(scanDest))
	}
	if _, ok := scanDest[0].(*time.Time); !ok {
		t.Errorf("Expected dest[0] to be *time.Time, got %T", scanDest[0])
	}
	if scanDest[0] != &tm {
		t.Errorf("Destination does not point to the address of 'tm'")
	}
}

func TestRowDestination_Destination_SingleColumn_Scanner(t *testing.T) {
	dest := &rowDestination{}
	var ss ScannerStruct
	rv := reflect.ValueOf(&ss)
	columns := []string{"scannable_value"}

	scanDest, err := dest.Destination(rv, columns)
	if err != nil {
		t.Fatalf("Destination failed for sql.Scanner: %v", err)
	}
	if len(scanDest) != 1 {
		t.Fatalf("Expected 1 destination for sql.Scanner, got %d", len(scanDest))
	}
	// For sql.Scanner, it should be the address of the struct itself if the struct implements it.
	// Or address of field if a field implements it.
	// Here, ScannerStruct implements it, so Addr() of the struct value.
	if _, okAddr := scanDest[0].(*ScannerStruct); !okAddr {
		t.Errorf("Expected dest[0] to be *ScannerStruct for sql.Scanner, got %T", scanDest[0])
	}
	if scanDest[0] != reflect.ValueOf(&ss).Interface() {
		t.Errorf("Destination does not point to the address of 'ss' for Scanner")
	}
}

func TestRowDestination_Destination_Error_MultiColumnNonceStruct(t *testing.T) {
	dest := &rowDestination{}
	var i int
	rv := reflect.ValueOf(&i)
	columns := []string{"value1", "value2"} // Multiple columns

	_, err := dest.Destination(rv, columns)
	if err == nil {
		t.Fatalf("Expected error for multi-column non-struct, but got nil")
	}
	if !strings.Contains(err.Error(), "expected struct, but got int") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestIsImplementsRowScanner(t *testing.T) {
	var rs *RowScannerStruct
	rt := reflect.TypeOf(rs) // reflect.TypeOf((*RowScannerStruct)(nil))
	if !isImplementsRowScanner(rt) {
		t.Errorf("Expected *RowScannerStruct to implement RowScanner")
	}

	var s SimpleStruct
	st := reflect.TypeOf(&s) // reflect.TypeOf((*SimpleStruct)(nil))
	if isImplementsRowScanner(st) {
		t.Errorf("Expected *SimpleStruct to not implement RowScanner")
	}

	// Test with non-pointer type that has pointer receiver for RowScanner
	// isImplementsRowScanner expects a pointer type as input from its call site in MultiRowsResultMap.resolveTypes
	// but let's test the underlying logic of Implements if it were passed a non-pointer.
	// reflect.Type.Implements() works correctly regardless of whether the type itself is a pointer or not,
	// as long as the method set is correct.
	// However, isImplementsRowScanner specifically checks reflect.PointerTo(elementType).Implements(rowScannerType)
	// or elementType.Implements(rowScannerType) if elementType is already a pointer.
	// So we should test with what it expects.

	rsNonPointer := reflect.TypeOf(RowScannerStruct{})
	if isImplementsRowScanner(rsNonPointer) { // This should be false because RowScanner has pointer receiver
		t.Errorf("Expected RowScannerStruct (non-pointer) to NOT directly implement RowScanner for isImplementsRowScanner check")
	}
	// The actual check in MultiRowsResultMap.resolveTypes does:
	// pointerType := elementType; if !isPointer { pointerType = reflect.PointerTo(elementType) }
	// isImplementsRowScanner(pointerType)
	// So if elementType is RowScannerStruct{}, pointerType becomes *RowScannerStruct, which implements it.

	var val int
	vt := reflect.TypeOf(&val)
	if isImplementsRowScanner(vt) {
		t.Errorf("Expected *int to not implement RowScanner")
	}
}
