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

// WalkByStep iterates over parts separated by the given separator without allocation.
// It calls the callback function for each non-empty part with its index.
// If the callback returns false, iteration stops.
//
// This is an allocation-free alternative to strings.Split + for range.
//
// Example:
//
//	WalkByStep("a.b.c", '.', func(i int, part string) bool {
//	    fmt.Println(i, part) // 0 a, 1 b, 2 c
//	    return true
//	})
func WalkByStep(s string, sep byte, fn func(index int, part string) bool) {
	start := 0
	i := 0
	for j := 0; j <= len(s); j++ {
		if j == len(s) || s[j] == sep {
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
