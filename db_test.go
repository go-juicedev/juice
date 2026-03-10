package juice

import "testing"

func TestDBManagerRegisteredReturnsClone(t *testing.T) {
	manager := &DBManager{}

	if err := manager.Add("primary", Source{}); err != nil {
		t.Fatalf("Add(primary) error = %v", err)
	}
	if err := manager.Add("secondary", Source{}); err != nil {
		t.Fatalf("Add(secondary) error = %v", err)
	}

	registered := manager.Registered()
	if len(registered) != 2 {
		t.Fatalf("Registered() len = %d, want 2", len(registered))
	}

	registered[0] = "mutated"

	afterMutation := manager.Registered()
	if afterMutation[0] != "primary" {
		t.Fatalf("Registered() leaked internal slice, got %q want %q", afterMutation[0], "primary")
	}
}
