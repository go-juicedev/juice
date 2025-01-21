package container

import (
	"testing"
)

func TestTrie_Basic(t *testing.T) {
	trie := NewTrie[string]()

	// Test empty trie
	if trie.Size() != 0 {
		t.Errorf("Expected empty trie size to be 0, got %d", trie.Size())
	}

	// Test insertion and retrieval
	testCases := []struct {
		key   string
		value string
	}{
		{"a", "value1"},
		{"ab", "value2"},
		{"abc", "value3"},
		{"b", "value4"},
		{"bc", "value5"},
	}

	for _, tc := range testCases {
		trie.Insert(tc.key, tc.value)
		if val, ok := trie.Get(tc.key); !ok || val != tc.value {
			t.Errorf("Get(%s) = (%s, %v), want (%s, true)", tc.key, val, ok, tc.value)
		}
	}

	// Test size
	if size := trie.Size(); size != len(testCases) {
		t.Errorf("Expected size %d, got %d", len(testCases), size)
	}

	// Test non-existent key
	if _, ok := trie.Get("nonexistent"); ok {
		t.Error("Expected Get of non-existent key to return false")
	}

	// Test empty key
	if _, ok := trie.Get(""); ok {
		t.Error("Expected Get of empty key to return false")
	}
}

func TestTrie_Delete(t *testing.T) {
	trie := NewTrie[string]()

	// Insert some values
	pairs := []struct {
		key   string
		value string
	}{
		{"test", "value1"},
		{"test.child", "value2"},
		{"test.child.grandchild", "value3"},
		{"other", "value4"},
	}

	for _, p := range pairs {
		trie.Insert(p.key, p.value)
	}

	// Test deletion
	testCases := []struct {
		key           string
		expectedFound bool
		description   string
	}{
		{"test.child", true, "existing key"},
		{"nonexistent", false, "non-existent key"},
		{"", false, "empty key"},
		{"test.child.grandchild", true, "leaf node"},
	}

	initialSize := trie.Size()

	for _, tc := range testCases {
		t.Run("Delete_"+tc.description, func(t *testing.T) {
			found := trie.Delete(tc.key)
			if found != tc.expectedFound {
				t.Errorf("Delete(%s) = %v, want %v", tc.key, found, tc.expectedFound)
			}

			// Verify key is actually deleted
			if _, exists := trie.Get(tc.key); exists {
				t.Errorf("Key %s still exists after deletion", tc.key)
			}
		})
	}

	// Verify size decreased correctly
	expectedDeleted := 2 // number of successful deletions
	if size := trie.Size(); size != initialSize-expectedDeleted {
		t.Errorf("Expected size %d, got %d", initialSize-expectedDeleted, size)
	}
}

func TestTrie_GetByPrefix(t *testing.T) {
	trie := NewTrie[string]()

	// Insert test data
	testData := []struct {
		key   string
		value string
	}{
		{"app.config", "config"},
		{"app.config.debug", "true"},
		{"app.config.port", "8080"},
		{"app.version", "1.0.0"},
		{"db.host", "localhost"},
		{"db.port", "5432"},
	}

	for _, td := range testData {
		trie.Insert(td.key, td.value)
	}

	// Test prefix search
	testCases := []struct {
		prefix        string
		expectedCount int
		expectedPairs map[string]string
		description   string
	}{
		{
			prefix:        "app.config",
			expectedCount: 3,
			expectedPairs: map[string]string{
				"app.config":       "config",
				"app.config.debug": "true",
				"app.config.port":  "8080",
			},
			description: "multiple matches",
		},
		{
			prefix:        "db",
			expectedCount: 2,
			expectedPairs: map[string]string{
				"db.host": "localhost",
				"db.port": "5432",
			},
			description: "partial prefix",
		},
		{
			prefix:        "nonexistent",
			expectedCount: 0,
			expectedPairs: map[string]string{},
			description:   "no matches",
		},
		{
			prefix:        "",
			expectedCount: 0,
			expectedPairs: map[string]string{},
			description:   "empty prefix",
		},
	}

	for _, tc := range testCases {
		t.Run("GetByPrefix_"+tc.description, func(t *testing.T) {
			results := trie.GetByPrefix(tc.prefix)

			if len(results) != tc.expectedCount {
				t.Errorf("GetByPrefix(%s) returned %d results, want %d",
					tc.prefix, len(results), tc.expectedCount)
			}

			// Verify each expected pair is in results
			for _, result := range results {
				expectedValue, exists := tc.expectedPairs[result.Key]
				if !exists {
					t.Errorf("Unexpected key in results: %s", result.Key)
					continue
				}
				if result.Value != expectedValue {
					t.Errorf("For key %s, got value %s, want %s",
						result.Key, result.Value, expectedValue)
				}
			}
		})
	}
}

func TestTrie_Overwrite(t *testing.T) {
	trie := NewTrie[string]()

	// Test overwriting values
	key := "test.key"
	original := "value1"
	updated := "value2"

	trie.Insert(key, original)
	if val, ok := trie.Get(key); !ok || val != original {
		t.Errorf("Expected initial value %s, got %s", original, val)
	}

	trie.Insert(key, updated)
	if val, ok := trie.Get(key); !ok || val != updated {
		t.Errorf("Expected updated value %s, got %s", updated, val)
	}

	// Size should remain 1 after overwrite
	if size := trie.Size(); size != 1 {
		t.Errorf("Expected size 1 after overwrite, got %d", size)
	}
}

func TestTrie_LongKeys(t *testing.T) {
	trie := NewTrie[string]()

	// Test very long key with many segments
	longKey := "this.is.a.very.long.key.with.many.segments"
	value := "test-value"

	trie.Insert(longKey, value)
	if val, ok := trie.Get(longKey); !ok || val != value {
		t.Errorf("Failed to retrieve value for long key")
	}

	// Test deleting segments of the long key
	segments := []string{
		"this.is.a.very.long.key.with.many",
		"this.is.a.very.long.key",
		"this.is.a",
		"this",
	}

	// Delete from longest to shortest
	for _, segment := range segments {
		if _, ok := trie.Get(segment); ok {
			t.Errorf("Unexpected value found for partial key %s", segment)
		}
	}

	// The original long key should still be retrievable
	if val, ok := trie.Get(longKey); !ok || val != value {
		t.Errorf("Long key value was lost after partial key checks")
	}
}

func BenchmarkTrie(b *testing.B) {
	trie := NewTrie[string]()
	testData := []struct {
		key   string
		value string
	}{
		{"app.config", "config"},
		{"app.config.debug", "true"},
		{"app.config.port", "8080"},
		{"app.version", "1.0.0"},
		{"db.host", "localhost"},
		{"db.port", "5432"},
		{"service.name", "myapp"},
		{"service.version", "2.0"},
		{"service.port", "9000"},
		{"cache.type", "redis"},
	}

	// Insert test data
	for _, td := range testData {
		trie.Insert(td.key, td.value)
	}

	b.Run("Insert", func(b *testing.B) {
		key := "test.benchmark.key"
		value := "benchmark-value"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			trie.Insert(key, value)
		}
	})

	b.Run("Get", func(b *testing.B) {
		key := "app.config.debug"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = trie.Get(key)
		}
	})

	b.Run("GetByPrefix", func(b *testing.B) {
		prefix := "app.config"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = trie.GetByPrefix(prefix)
		}
	})

	b.Run("Delete", func(b *testing.B) {
		key := "test.benchmark.key"
		value := "benchmark-value"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			trie.Insert(key, value)
			_ = trie.Delete(key)
		}
	})
}
