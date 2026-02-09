package reflectlite

import (
	"reflect"
	"testing"
)

func TestTypeIdentify_BasicType_type_test(t *testing.T) {
	result := TypeIdentify[int]()
	if result != "int" {
		t.Errorf("Expected 'int', got '%s'", result)
	}
}

func TestTypeIdentify_StructType_type_test(t *testing.T) {
	type testType struct {
		field string // nolint:unused
	}
	result := TypeIdentify[testType]()
	if result != "github.com/go-juicedev/juice/internal/reflectlite.testType" {
		t.Errorf("Expected 'reflectlite.testType', got '%s'", result)
	}
}

func TestTypeIdentify_SliceType_type_test(t *testing.T) {
	type testType []int
	result := TypeIdentify[testType]()
	if result != "slice[int]" {
		t.Errorf("Expected 'slice[int]', got '%s'", result)
	}
}

func TestTypeIdentify_MapType_type_test(t *testing.T) {
	type testType map[string]int
	result := TypeIdentify[testType]()
	if result != "map[string]int" {
		t.Errorf("Expected 'map[string]int', got '%s'", result)
	}
}

func TestTypeIdentify_PointerType_type_test(t *testing.T) {
	type testType *int
	result := TypeIdentify[testType]()
	if result != "ptr[int]" {
		t.Errorf("Expected 'ptr[int]', got '%s'", result)
	}
}

func TestTypeIdentify_AnonymousStruct_type_test(t *testing.T) {
	type testType struct {
		field string // nolint:unused
	}
	result := TypeIdentify[struct{ testType }]()
	if result != "struct { reflectlite.testType }" {
		t.Errorf("Expected 'struct { reflectlite.testType }', got '%s'", result)
	}
}

func TestType_Indirect_type_test(t *testing.T) {
	type myInt int
	type ptrMyInt *myInt

	var i myInt = 10
	var p ptrMyInt = &i

	// Test with a non-pointer type
	rtNonPtr := reflect.TypeOf(i)
	typeWrapperNonPtr := TypeFrom(rtNonPtr)
	indirectTypeWrapperNonPtr := typeWrapperNonPtr.Indirect() // Call first time
	if indirectTypeWrapperNonPtr.Type != rtNonPtr {
		t.Errorf("Expected indirect type to be '%s', got '%s'", rtNonPtr.String(), indirectTypeWrapperNonPtr.String())
	}
	if !typeWrapperNonPtr.indirectTypeSet || typeWrapperNonPtr.indirectType != rtNonPtr {
		t.Errorf("Cache not set correctly for non-pointer type. Set: %v, Cached: %s", typeWrapperNonPtr.indirectTypeSet, typeWrapperNonPtr.indirectType)
	}
	_ = typeWrapperNonPtr.Indirect() // Call second time to check cache usage (implicitly)

	// Test with a pointer type
	rtPtr := reflect.TypeOf(p)
	typeWrapperPtr := TypeFrom(rtPtr)
	indirectTypeWrapperPtr := typeWrapperPtr.Indirect() // Call first time
	if indirectTypeWrapperPtr.Type != rtNonPtr {        // Should be myInt, not *myInt
		t.Errorf("Expected indirect type to be '%s', got '%s'", rtNonPtr.String(), indirectTypeWrapperPtr.String())
	}
	if !typeWrapperPtr.indirectTypeSet || typeWrapperPtr.indirectType != rtNonPtr {
		t.Errorf("Cache not set correctly for pointer type. Set: %v, Cached: %s", typeWrapperPtr.indirectTypeSet, typeWrapperPtr.indirectType)
	}
	cachedIndirect := typeWrapperPtr.Indirect() // Call second time
	if cachedIndirect.Type != rtNonPtr {
		t.Errorf("Expected cached indirect type to be '%s', got '%s'", rtNonPtr.String(), cachedIndirect.String())
	}

	// Test with nil type (should not panic)
	var nilType reflect.Type
	nilTypeWrapper := TypeFrom(nilType)
	indirectNil := nilTypeWrapper.Indirect()
	if indirectNil.Type != nil {
		t.Errorf("Expected nil type for indirect of nil, got %v", indirectNil.Type)
	}
}

func TestType_GetFieldIndexesFromTag_Caching_type_test(t *testing.T) {
	type CacheStruct struct {
		FieldA string `testtag:"field_a"`
		FieldB int    `testtag:"field_b"`
	}
	cs := CacheStruct{}
	rt := reflect.TypeOf(cs)
	typeWrapper := TypeFrom(rt)

	// Call first time to populate cache
	indexesA1, foundA1 := typeWrapper.GetFieldIndexesFromTag("testtag", "field_a")
	if !foundA1 || len(indexesA1) != 1 || indexesA1[0] != 0 {
		t.Fatalf("Initial call for FieldA failed. Found: %v, Indexes: %v", foundA1, indexesA1)
	}

	// Call second time, should use cache
	// For unit testing, we can't directly check if sync.Map was hit without mocks or specific counters.
	// We rely on the correctness of sync.Map and our caching logic.
	// The main test is that the result remains correct.
	indexesA2, foundA2 := typeWrapper.GetFieldIndexesFromTag("testtag", "field_a")
	if !foundA2 || len(indexesA2) != 1 || indexesA2[0] != 0 {
		t.Errorf("Second call for FieldA failed or gave different result. Found: %v, Indexes: %v", foundA2, indexesA2)
	}

	indexesB1, foundB1 := typeWrapper.GetFieldIndexesFromTag("testtag", "field_b")
	if !foundB1 || len(indexesB1) != 1 || indexesB1[0] != 1 {
		t.Fatalf("Initial call for FieldB failed. Found: %v, Indexes: %v", foundB1, indexesB1)
	}
	indexesB2, foundB2 := typeWrapper.GetFieldIndexesFromTag("testtag", "field_b")
	if !foundB2 || len(indexesB2) != 1 || indexesB2[0] != 1 {
		t.Errorf("Second call for FieldB failed or gave different result. Found: %v, Indexes: %v", foundB2, indexesB2)
	}

	// Test not found caching
	_, foundNF1 := typeWrapper.GetFieldIndexesFromTag("testtag", "non_existent")
	if foundNF1 {
		t.Error("Expected 'non_existent' to not be found on first call")
	}
	_, foundNF2 := typeWrapper.GetFieldIndexesFromTag("testtag", "non_existent")
	if foundNF2 {
		t.Error("Expected 'non_existent' to not be found on second call (cached)")
	}
}

func TestType_GetFieldIndexesFromTag_NestedAndAnonymous_type_test(t *testing.T) {
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
	type PtrOuter struct {
		Ptr *Outer `tag:"ptr_outer"`
	}

	outer := Outer{}
	rtOuter := reflect.TypeOf(outer)
	typeWrapperOuter := TypeFrom(rtOuter)

	// Test direct field on Outer
	idxOuter, okOuter := typeWrapperOuter.GetFieldIndexesFromTag("tag", "outer")
	if !okOuter || len(idxOuter) != 1 || idxOuter[0] != 1 {
		t.Errorf("OuterField: Expected [1], got %v (ok: %v)", idxOuter, okOuter)
	}

	// Test field in embedded Inner struct
	idxMid, okMid := typeWrapperOuter.GetFieldIndexesFromTag("tag", "mid")
	if !okMid || len(idxMid) != 2 || idxMid[0] != 0 || idxMid[1] != 1 { // Outer.Inner.MidField
		t.Errorf("MidField: Expected [0 1], got %v (ok: %v)", idxMid, okMid)
	}

	// Test field in deeply embedded InnerMost struct
	idxDeep, okDeep := typeWrapperOuter.GetFieldIndexesFromTag("tag", "deep")
	if !okDeep || len(idxDeep) != 3 || idxDeep[0] != 0 || idxDeep[1] != 0 || idxDeep[2] != 0 { // Outer.Inner.InnerMost.DeepField
		t.Errorf("DeepField: Expected [0 0 0], got %v (ok: %v)", idxDeep, okDeep)
	}

	// Test with pointer to struct
	ptrOuterType := reflect.TypeOf(PtrOuter{})
	typeWrapperPtrOuter := TypeFrom(ptrOuterType)
	idxPtr, okPtr := typeWrapperPtrOuter.GetFieldIndexesFromTag("tag", "ptr_outer")
	if !okPtr || len(idxPtr) != 1 || idxPtr[0] != 0 {
		t.Errorf("PtrOuter.Ptr: Expected [0], got %v (ok: %v)", idxPtr, okPtr)
	}

	// Test tag on field that is a pointer to a struct, looking for a tag within that pointed-to struct
	// This case should effectively be handled by GetFieldIndexesFromTag on the type of the field Ptr *Outer
	// The current GetFieldIndexesFromTag operates on the type it's called on.
	// To test finding "deep" from PtrOuter, one would get the field Ptr, get its type, and then call GetFieldIndexesFromTag.
	// The current test structure for GetFieldIndexesFromTag is correct for its intended use.
}

func TestTypeIdentify_MoreComplexTypes_type_test(t *testing.T) {
	tests := []struct {
		name     string
		typeOf   func() reflect.Type
		expected string
	}{
		{
			name:     "map[string]*struct{f int}",
			typeOf:   func() reflect.Type { type T map[string]*struct{ f int }; return reflect.TypeOf((*T)(nil)).Elem() },
			expected: "map[string]ptr[struct { f int }]",
		},
		{
			name:     "[]*map[int]string",
			typeOf:   func() reflect.Type { type T []*map[int]string; return reflect.TypeOf((*T)(nil)).Elem() },
			expected: "slice[ptr[map[int]string]]",
		},
		{
			name:     "chan struct{}",
			typeOf:   func() reflect.Type { type T chan struct{}; return reflect.TypeOf((*T)(nil)).Elem() },
			expected: "chan[struct {}]",
		},
		{
			name:     "func() error",
			typeOf:   func() reflect.Type { type T func() error; return reflect.TypeOf((*T)(nil)).Elem() },
			expected: "github.com/go-juicedev/juice/internal/reflectlite.T",
		},
		{
			name:     "interface{}",
			typeOf:   func() reflect.Type { return reflect.TypeOf((*any)(nil)).Elem() },
			expected: "interface {}", // Or "any" depending on Go version and exact stdlib representation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := typeToString(tt.typeOf())
			// For interface{} vs any, accept both as Go evolves
			if tt.expected == "interface {}" && result == "any" {
				// this is fine
			} else if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
