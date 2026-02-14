/*
Copyright 2025 eatmoreapple

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

package stringutil

import (
	"strings"
	"testing"
)

// testStrings for benchmarking
var testStrings = []string{
	"a",
	"a.b",
	"a.b.c",
	"com.example.user.model",
	"com.example.user.repository.UserRepository",
	"very.long.namespace.with.many.parts.for.testing.purposes",
}

// inlineSplit simulates the current inline implementation
func inlineSplit(s string, fn func(index int, part string) bool) {
	start := 0
	i := 0
	for j := 0; j <= len(s); j++ {
		if j == len(s) || s[j] == '.' {
			if j > start {
				if !fn(i, s[start:j]) {
					return
				}
				i++
			}
			start = j + 1
		}
	}
}

// BenchmarkStringsSplit benchmarks the standard strings.Split approach
func BenchmarkStringsSplit(b *testing.B) {
	for _, s := range testStrings {
		b.Run(s, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				parts := strings.Split(s, ".")
				for idx, part := range parts {
					_ = idx
					_ = part
				}
			}
		})
	}
}

// BenchmarkInlineSplit benchmarks the inline implementation (current approach)
func BenchmarkInlineSplit(b *testing.B) {
	for _, s := range testStrings {
		b.Run(s, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				inlineSplit(s, func(idx int, part string) bool {
					_ = idx
					_ = part
					return true
				})
			}
		})
	}
}

// BenchmarkWalkByStep benchmarks the WalkByStep function
func BenchmarkWalkByStep(b *testing.B) {
	for _, s := range testStrings {
		b.Run(s, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				WalkByStep(s, '.', func(idx int, part string) bool {
					_ = idx
					_ = part
					return true
				})
			}
		})
	}
}

// BenchmarkStringsSplitWithWork benchmarks strings.Split with some work in the loop
func BenchmarkStringsSplitWithWork(b *testing.B) {
	for _, s := range testStrings {
		b.Run(s, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				parts := strings.Split(s, ".")
				for _, part := range parts {
					// Simulate some work (e.g., hash computation)
					_ = len(part) * len(part)
				}
			}
		})
	}
}

// BenchmarkInlineSplitWithWork benchmarks inline split with some work in the loop
func BenchmarkInlineSplitWithWork(b *testing.B) {
	for _, s := range testStrings {
		b.Run(s, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				inlineSplit(s, func(idx int, part string) bool {
					// Simulate some work (e.g., hash computation)
					_ = len(part) * len(part)
					return true
				})
			}
		})
	}
}

// BenchmarkWalkByStepWithWork benchmarks WalkByStep with some work in the loop
func BenchmarkWalkByStepWithWork(b *testing.B) {
	for _, s := range testStrings {
		b.Run(s, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				WalkByStep(s, '.', func(idx int, part string) bool {
					// Simulate some work (e.g., hash computation)
					_ = len(part) * len(part)
					return true
				})
			}
		})
	}
}

// BenchmarkCollectParts compares collecting parts into a slice
func BenchmarkCollectParts(b *testing.B) {
	b.Run("StringsSplit", func(b *testing.B) {
		s := "com.example.user.repository.UserRepository"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = strings.Split(s, ".")
		}
	})

	b.Run("WalkByStep", func(b *testing.B) {
		s := "com.example.user.repository.UserRepository"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var parts []string
			WalkByStep(s, '.', func(idx int, part string) bool {
				parts = append(parts, part)
				return true
			})
			_ = parts
		}
	})
}
