package reflectlite

import (
	"reflect"
	"slices"
	"testing"
)

func TestIndirectType_type_test(t *testing.T) {
	type myInt int

	tests := []struct {
		name string
		in   reflect.Type
		want reflect.Type
	}{
		{"non-pointer", reflect.TypeFor[myInt](), reflect.TypeFor[myInt]()},
		{"pointer", reflect.TypeFor[*myInt](), reflect.TypeFor[myInt]()},
		{"double pointer", reflect.TypeFor[**myInt](), reflect.TypeFor[myInt]()},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IndirectType(tt.in); got != tt.want {
				t.Errorf("IndirectType(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestLookupFieldIndexByTag_type_test(t *testing.T) {
	type InnerMost struct {
		DeepField string `tag:"deep"`
	}
	type Inner struct {
		InnerMost     // Anonymous
		MidField  int `tag:"mid"`
	}
	type Outer struct {
		Inner           // Anonymous
		OuterField bool `tag:"outer"`
	}

	rtOuter := reflect.TypeFor[Outer]()

	tests := []struct {
		name  string
		rType reflect.Type
		value string
		want  []int
		ok    bool
	}{
		{"direct field", rtOuter, "outer", []int{1}, true},
		{"embedded field", rtOuter, "mid", []int{0, 1}, true},
		{"deeply embedded field", rtOuter, "deep", []int{0, 0, 0}, true},
		{"pointer to struct", reflect.TypeFor[*Outer](), "outer", []int{1}, true},
		{"not found", rtOuter, "nope", nil, false},
		{"non-struct", reflect.TypeFor[int](), "outer", nil, false},
		{"nil type", nil, "outer", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := LookupFieldIndexByTag(tt.rType, "tag", tt.value)
			if ok != tt.ok {
				t.Fatalf("LookupFieldIndexByTag(%v, tag, %q) ok = %v, want %v", tt.rType, tt.value, ok, tt.ok)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("LookupFieldIndexByTag(%v, tag, %q) = %v, want %v", tt.rType, tt.value, got, tt.want)
			}
		})
	}
}
