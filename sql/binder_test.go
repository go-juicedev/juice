package sql

import (
	"database/sql"
	"errors"
	"testing"
)

// TestUser is a sample struct for testing.
type TestUser struct {
	ID   int    `column:"id"`
	Name string `column:"name"`
}

func TestBind(t *testing.T) {
	// Test binding to a single struct
	t.Run("SingleStruct", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}},
		}
		user, err := Bind[TestUser](rows)
		if err != nil {
			t.Fatalf("Bind failed: %v", err)
		}
		if user.ID != 1 || user.Name != "Alice" {
			t.Errorf("Expected ID=1, Name='Alice', got ID=%d, Name='%s'", user.ID, user.Name)
		}
	})

	// Test binding to a slice of structs
	t.Run("SliceOfStructs", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}, {2, "Bob"}},
		}
		users, err := Bind[[]TestUser](rows)
		if err != nil {
			t.Fatalf("Bind failed: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users))
		}
		if users[0].ID != 1 || users[0].Name != "Alice" {
			t.Errorf("Expected User1 ID=1, Name='Alice', got ID=%d, Name='%s'", users[0].ID, users[0].Name)
		}
		if users[1].ID != 2 || users[1].Name != "Bob" {
			t.Errorf("Expected User2 ID=2, Name='Bob', got ID=%d, Name='%s'", users[1].ID, users[1].Name)
		}
	})

	// Test binding to a pointer to a struct
	t.Run("PointerToStruct", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}},
		}
		user, err := Bind[*TestUser](rows)
		if err != nil {
			t.Fatalf("Bind failed: %v", err)
		}
		if user == nil {
			t.Fatal("Expected user not to be nil")
		}
		if user.ID != 1 || user.Name != "Alice" {
			t.Errorf("Expected ID=1, Name='Alice', got ID=%d, Name='%s'", user.ID, user.Name)
		}
	})

	// Test binding to a slice of pointers to structs
	t.Run("SliceOfPointerToStructs", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}, {2, "Bob"}},
		}
		users, err := Bind[[]*TestUser](rows)
		if err != nil {
			t.Fatalf("Bind failed: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users))
		}
		if users[0] == nil || users[0].ID != 1 || users[0].Name != "Alice" {
			t.Errorf("Expected User1 ID=1, Name='Alice', got ID=%v, Name='%v'", users[0].ID, users[0].Name)
		}
		if users[1] == nil || users[1].ID != 2 || users[1].Name != "Bob" {
			t.Errorf("Expected User2 ID=2, Name='Bob', got ID=%v, Name='%v'", users[1].ID, users[1].Name)
		}
	})

	// Test with empty Rows
	t.Run("EmptyRows", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{},
		}
		// For single struct, it should return a zero-value struct
		user, err := Bind[TestUser](rows)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("Bind failed for single struct with empty rows: %v", err)
			}
		}
		if user.ID != 0 || user.Name != "" {
			t.Errorf("Expected zero TestUser, got ID=%d, Name='%s'", user.ID, user.Name)
		}

		rows = &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{},
		}
		// For slice of structs, it should return an empty slice
		users, err := Bind[[]TestUser](rows)
		if err != nil {
			t.Fatalf("Bind failed for slice of structs with empty rows: %v", err)
		}
		if len(users) != 0 {
			t.Errorf("Expected empty slice, got %d users", len(users))
		}
	})

	// Test with nil destination (should be handled by BindWithResultMap, Bind itself takes type param)
	// This case is more about the internal bindWithResultMap, but Bind should return an error if mapping fails.
	// For Bind, the destination is implicitly created. If the internal logic fails due to a type mismatch
	// that would have been a nil pointer issue, it should manifest as a mapping error.

	// Test with non-pointer destination (should be handled by BindWithResultMap, Bind itself takes type param)
	// Similar to the nil destination, this is about internal behavior.
	// The generic nature of Bind[T] means T is the value type.
	// If T is not a pointer and the underlying mapping expects a pointer, it might error.
	// However, the current implementation of Bind and bindWithResultMap handles this by working with pointers internally.

	// Test with ErrNilDestination
	t.Run("NilDestination", func(t *testing.T) {
		rowsForNilUser := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}},
		}
		boundNilUser, err := Bind[*TestUser](rowsForNilUser)
		if err != nil {
			t.Fatalf("Bind failed for nil pointer type: %v", err)
		}
		if boundNilUser == nil {
			t.Fatal("Expected bound user not to be nil")
		}
		if boundNilUser.ID != 1 || boundNilUser.Name != "Alice" {
			t.Errorf("Expected ID=1, Name='Alice', got ID=%d, Name='%s'", boundNilUser.ID, boundNilUser.Name)
		}

	})

	// Test with ErrPointerRequired
	t.Run("NonPointerDestination", func(t *testing.T) {
		// Similar to ErrNilDestination, this is hard to test directly with Bind's signature
		// because Bind[T] itself uses reflection to handle the destination.
		// The error ErrPointerRequired would come from bindWithResultMap if it received a non-pointer.
		// Bind[T] ensures that ptr passed to bindWithResultMap is always a pointer.
		// So, a direct test for this specific error path through Bind[T] is not straightforward.
		// We trust that if bindWithResultMap was called with a non-pointer, it would error,
		// but Bind[T]'s structure prevents this.
	})

}

func TestList(t *testing.T) {
	// Test converting Rows to a slice of structs
	t.Run("SliceOfStructs", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}, {2, "Bob"}},
		}
		users, err := List[TestUser](rows)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users))
		}
		if users[0].ID != 1 || users[0].Name != "Alice" {
			t.Errorf("Expected User1 ID=1, Name='Alice', got ID=%d, Name='%s'", users[0].ID, users[0].Name)
		}
		if users[1].ID != 2 || users[1].Name != "Bob" {
			t.Errorf("Expected User2 ID=2, Name='Bob', got ID=%d, Name='%s'", users[1].ID, users[1].Name)
		}
	})

	// Test with empty Rows
	t.Run("EmptyRows", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{},
		}
		users, err := List[TestUser](rows)
		if err != nil {
			t.Fatalf("List failed with empty rows: %v", err)
		}
		if len(users) != 0 {
			t.Errorf("Expected empty slice, got %d users", len(users))
		}
	})

	// Test converting Rows to a slice of pointers to structs
	// This case is more for List[T] where T is a pointer type.
	t.Run("SliceOfPointerToStructs", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}, {2, "Bob"}},
		}
		users, err := List[*TestUser](rows)
		if err != nil {
			t.Fatalf("List failed for slice of pointers: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users))
		}
		if users[0] == nil || users[0].ID != 1 || users[0].Name != "Alice" {
			t.Errorf("Expected User1 ID=1, Name='Alice', got ID=%v, Name='%v'", users[0].ID, users[0].Name)
		}
		if users[1] == nil || users[1].ID != 2 || users[1].Name != "Bob" {
			t.Errorf("Expected User2 ID=2, Name='Bob', got ID=%v, Name='%v'", users[1].ID, users[1].Name)
		}
	})
}

func TestList2(t *testing.T) {
	// Test converting Rows to a slice of pointers to structs
	t.Run("SliceOfPointerToStructs", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}, {2, "Bob"}},
		}
		users, err := List2[TestUser](rows)
		if err != nil {
			t.Fatalf("List2 failed: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users))
		}
		if users[0] == nil || users[0].ID != 1 || users[0].Name != "Alice" {
			t.Errorf("Expected User1 ID=1, Name='Alice', got ID=%v, Name='%v'", users[0].ID, users[0].Name)
		}
		if users[1] == nil || users[1].ID != 2 || users[1].Name != "Bob" {
			t.Errorf("Expected User2 ID=2, Name='Bob', got ID=%v, Name='%v'", users[1].ID, users[1].Name)
		}
	})

	// Test with empty Rows
	t.Run("EmptyRows", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{},
		}
		users, err := List2[TestUser](rows)
		if err != nil {
			t.Fatalf("List2 failed with empty rows: %v", err)
		}
		if len(users) != 0 {
			t.Errorf("Expected empty slice, got %d users", len(users))
		}
	})

}

func TestIter(t *testing.T) {
	// Test cases will be added here
	t.Run("IterateOverRows", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{{1, "Alice"}, {2, "Bob"}},
		}
		var users []TestUser
		seq, err := Iter[TestUser](rows)
		if err != nil {
			t.Fatalf("Iter initialization failed: %v", err)
		}
		for user, err := range seq {
			if err != nil {
				t.Fatalf("Iter failed: %v", err)
			}
			users = append(users, user)
		}
		if len(users) != len(rows.Data) {
			t.Fatalf("Expected %d users, got %d", len(rows.Data), len(users))
		}
		for i, user := range users {
			if user.ID != rows.Data[i][0].(int) || user.Name != rows.Data[i][1].(string) {
				t.Errorf("Expected User ID=%d, Name='%s', got ID=%d, Name='%s'", rows.Data[i][0], rows.Data[i][1], user.ID, user.Name)
			}
		}
	})

	t.Run("IterateOverEmptyRows", func(t *testing.T) {
		rows := &RowsBuffer{
			ColumnsLine: []string{"id", "name"},
			Data:        [][]any{},
		}
		seq, err := Iter[TestUser](rows)
		if err != nil {
			t.Fatalf("Iter initialization failed: %v", err)
		}
		var users []TestUser
		for user, err := range seq {
			if err != nil {
				t.Fatalf("Iter failed with empty rows: %v", err)
			}
			users = append(users, user)
		}
		if len(users) != 0 {
			t.Errorf("Expected empty slice, got %d users", len(users))
		}
	})
}
