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

package container

import (
	"sort"
)

// TrieNode represents a node in the trie
type TrieNode[T any] struct {
	part     string
	children []*TrieNode[T]
	value    T
	hasValue bool
}

// Trie implements a prefix tree optimized for memory usage with shared prefixes
type Trie[T any] struct {
	root *TrieNode[T]
	size int
}

// NewTrie creates a new Trie instance
func NewTrie[T any]() *Trie[T] {
	return &Trie[T]{
		root: &TrieNode[T]{
			children: make([]*TrieNode[T], 0, 4),
		},
	}
}

// findChild performs binary search to find a child node by part
func (n *TrieNode[T]) findChild(part string) (int, bool) {
	if len(n.children) == 0 {
		return 0, false
	}
	i := sort.Search(len(n.children), func(i int) bool {
		return n.children[i].part >= part
	})
	if i < len(n.children) && n.children[i].part == part {
		return i, true
	}
	return i, false
}

// insertChild inserts a child node while maintaining sorted order
func (n *TrieNode[T]) insertChild(child *TrieNode[T]) {
	idx, found := n.findChild(child.part)
	if found {
		n.children[idx] = child // Replace existing node
		return
	}

	// Insert at the correct position
	n.children = append(n.children, nil)
	copy(n.children[idx+1:], n.children[idx:])
	n.children[idx] = child
}

// Insert adds or updates a value in the trie
// Time complexity: O(k * log n) where k is the number of parts in the key
// and n is the average number of children per node
func (t *Trie[T]) Insert(key string, value T) {
	if key == "" {
		return
	}

	current := t.root

	// Iterate through the key character by character to avoid splitting into parts
	start := 0
	for i := 0; i <= len(key); i++ {
		if i == len(key) || key[i] == '.' {
			// Extract the part between start and i-1
			if i > start { // avoid empty parts
				part := key[start:i]
				idx, found := current.findChild(part)
				if !found {
					// Pre-allocate space for children to reduce reallocations
					node := &TrieNode[T]{
						part:     part,
						children: make([]*TrieNode[T], 0, 4), // Start with capacity 4
					}
					current.insertChild(node)
					idx, _ = current.findChild(part)
				}
				current = current.children[idx]
			}
			start = i + 1
		}
	}

	if !current.hasValue {
		t.size++
	}
	current.value = value
	current.hasValue = true
}

// Get retrieves a value from the trie
// Time complexity: O(k * log n) where k is the number of parts in the key
// and n is the average number of children per node
func (t *Trie[T]) Get(key string) (T, bool) {
	if key == "" {
		var zero T
		return zero, false
	}

	current := t.root

	// Iterate through the key character by character to avoid splitting into parts
	start := 0
	for i := 0; i <= len(key); i++ {
		if i == len(key) || key[i] == '.' {
			// Extract the part between start and i-1
			if i > start { // avoid empty parts
				part := key[start:i]
				idx, found := current.findChild(part)
				if !found {
					var zero T
					return zero, false
				}
				current = current.children[idx]
			}
			start = i + 1
		}
	}

	if !current.hasValue {
		var zero T
		return zero, false
	}
	return current.value, true
}

// removeChild removes a child node at the specified index
func (n *TrieNode[T]) removeChild(idx int) {
	copy(n.children[idx:], n.children[idx+1:])
	n.children = n.children[:len(n.children)-1]
}

// Delete removes a key-value pair from the trie
// Time complexity: O(k * log n) where k is the number of parts in the key
// and n is the average number of children per node
func (t *Trie[T]) Delete(key string) bool {
	if key == "" {
		return false
	}

	current := t.root
	var nodes []*TrieNode[T]
	var indices []int

	// Pre-allocate with initial capacity
	nodes = append(nodes, current)

	// Find the node and collect path by iterating through the key character by character
	start := 0
	for i := 0; i <= len(key); i++ {
		if i == len(key) || key[i] == '.' {
			// Extract the part between start and i-1
			if i > start { // avoid empty parts
				part := key[start:i]
				idx, found := current.findChild(part)
				if !found {
					return false
				}
				indices = append(indices, idx)
				current = current.children[idx]
				nodes = append(nodes, current)
			}
			start = i + 1
		}
	}

	if !current.hasValue {
		return false
	}

	// Mark as deleted and update size
	current.hasValue = false
	t.size--

	// Clean up unused nodes from bottom to top
	for i := len(nodes) - 1; i > 0; i-- {
		node := nodes[i]
		parent := nodes[i-1]
		idx := indices[i-1]

		if !node.hasValue && len(node.children) == 0 {
			parent.removeChild(idx)
		} else {
			break // Stop if we find a node that should be kept
		}
	}

	return true
}

// Size returns the number of key-value pairs in the trie
func (t *Trie[T]) Size() int {
	return t.size
}

// KeyValue represents a key-value pair in the trie
type KeyValue[T any] struct {
	Key   string
	Value T
}

// collectValues recursively collects all key-value pairs under a node
func (t *Trie[T]) collectValues(node *TrieNode[T], prefix string, result *[]KeyValue[T]) {
	// If current node has a value, add it to results
	if node.hasValue {
		*result = append(*result, KeyValue[T]{
			Key:   prefix,
			Value: node.value,
		})
	}

	// Recursively collect values from all children
	for _, child := range node.children {
		childPrefix := prefix
		if childPrefix != "" {
			childPrefix += "."
		}
		t.collectValues(child, childPrefix+child.part, result)
	}
}

// GetByPrefix returns all key-value pairs with the given prefix
// Time complexity: O(k * log n + m) where k is the number of parts in the prefix,
// n is the average number of children per node, and m is the number of matching nodes
func (t *Trie[T]) GetByPrefix(prefix string) []KeyValue[T] {
	if prefix == "" {
		return nil
	}

	current := t.root

	// Navigate to the prefix node by iterating through the prefix character by character
	start := 0
	for i := 0; i <= len(prefix); i++ {
		if i == len(prefix) || prefix[i] == '.' {
			// Extract the part between start and i-1
			if i > start { // avoid empty parts
				part := prefix[start:i]
				idx, found := current.findChild(part)
				if !found {
					return nil
				}
				current = current.children[idx]
			}
			start = i + 1
		}
	}

	// Pre-allocate result slice with a reasonable capacity
	result := make([]KeyValue[T], 0, 8)
	t.collectValues(current, prefix, &result)
	return result
}
