package reflectlite

import (
	"reflect"
	"testing"
)

func TestValue_FindFieldFromTag_value_test(t *testing.T) {
	type A struct {
		AName string `param:"a_name"`
	}

	type B struct {
		BName string `param:"b_name"`
		A
	}

	var b B

	b.AName = "a_name"
	b.BName = "b_name"

	value := ValueFrom(reflect.ValueOf(b))

	v, ok := value.FindFieldFromTag("param", "a_name")
	if !ok || !v.IsValid() {
		t.Error("expect a_name, but not found or invalid")
	}
	if v.String() != "a_name" {
		t.Error("expect a_name")
	}
}

func TestValue_GetFieldIndexesFromTag_value_test(t *testing.T) {
	type A struct {
		AName string `param:"a_name"`
	}

	type B struct {
		BName string `param:"b_name"`
		A
	}

	var b B

	b.AName = "a_name"
	b.BName = "b_name"

	value := ValueFrom(reflect.ValueOf(b))

	// Test finding field index by tag
	indexes, ok := value.GetFieldIndexesFromTag("param", "a_name")
	if !ok {
		t.Error("expected to find a_name")
	}
	if len(indexes) != 2 || indexes[0] != 1 || indexes[1] != 0 {
		t.Errorf("expected indexes [1 0], got %v", indexes)
	}
	t.Log(indexes)

	// Test not finding field index by non-existent tag
	indexes, ok = value.GetFieldIndexesFromTag("param", "non_existent")
	if ok {
		t.Error("expected not to find non_existent")
	}
	if indexes != nil {
		t.Errorf("expected nil indexes, got %v", indexes)
	}

	// Test not finding field index in non-struct type
	nonStructValue := ValueFrom(reflect.ValueOf("string"))
	indexes, ok = nonStructValue.GetFieldIndexesFromTag("param", "a_name")
	if ok {
		t.Error("expected not to find a_name in non-struct type")
	}
	if indexes != nil {
		t.Errorf("expected nil indexes, got %v", indexes)
	}
}

func TestValue_Unwrap_value_test(t *testing.T) {
	s := "hello"
	ps := &s
	pps := &ps

	valPps := ValueOf(pps)

	// Test unwrapping multiple pointers
	unwrapped1 := valPps.Unwrap()
	if unwrapped1.String() != "hello" {
		t.Errorf("Expected unwrapped1 to be 'hello', got '%s'", unwrapped1.String())
	}

	// Test with non-pointer
	valS := ValueOf(s)
	unwrappedS := valS.Unwrap()
	if unwrappedS.String() != "hello" {
		t.Errorf("Expected unwrappedS to be 'hello', got '%s'", unwrappedS.String())
	}

	// Test with nil pointer
	var nilStr *string
	valNil := ValueOf(nilStr)
	unwrappedNil := valNil.Unwrap()
	if unwrappedNil.IsValid() && !unwrappedNil.IsNil() {
		t.Errorf("Expected unwrapped nil to be nil, got valid non-nil: %v", unwrappedNil)
	}
}

func TestValue_IndirectType_value_test(t *testing.T) {
	s := "world"
	ps := &s

	valPs := ValueOf(ps)
	expectedType := reflect.TypeOf(s)

	// Test IndirectType
	type1 := valPs.IndirectType()
	if type1.Type != expectedType {
		t.Errorf("Expected IndirectType to be '%s', got '%s'", expectedType.String(), type1.String())
	}
}

func TestValue_IndirectKind_value_test(t *testing.T) {
	i := 123
	pi := &i
	valPi := ValueOf(pi)
	expectedKind := reflect.Int

	// Test IndirectKind
	kind1 := valPi.IndirectKind()
	if kind1 != expectedKind {
		t.Errorf("Expected IndirectKind to be '%s', got '%s'", expectedKind, kind1)
	}
}

func TestValue_FindFieldFromTag_MoreScenarios_value_test(t *testing.T) {
	type InnerMost struct {
		DeepField string `tag:"deep"`
	}
	type Inner struct {
		InnerMost            // Anonymous
		MidField  int        `tag:"mid"`
		MidPtr    *InnerMost `tag:"mid_ptr"`
	}
	type Outer struct {
		InnerField Inner  `tag:"inner_field"`
		OuterField bool   `tag:"outer"`
		OuterPtr   *Inner `tag:"outer_ptr"`
	}

	im := InnerMost{DeepField: "deep_val"}
	in := Inner{InnerMost: im, MidField: 10, MidPtr: &im}
	instance := Outer{InnerField: in, OuterField: true, OuterPtr: &in}
	val := ValueOf(instance)

	// Direct field
	fieldOuter, okOuter := val.FindFieldFromTag("tag", "outer")
	if !okOuter || !fieldOuter.IsValid() || fieldOuter.Bool() != true {
		t.Errorf("OuterField: Expected true, ok: %v, val: %v", okOuter, fieldOuter)
	}

	// Nested field
	// To find "mid", we need to get "inner_field" first, then call FindFieldFromTag on it
	innerFieldValue, okInner := val.FindFieldFromTag("tag", "inner_field")
	if !okInner || !innerFieldValue.IsValid() {
		t.Fatalf("Could not find 'inner_field'")
	}
	fieldMid, okMid := innerFieldValue.FindFieldFromTag("tag", "mid")
	if !okMid || !fieldMid.IsValid() || fieldMid.Int() != 10 {
		t.Errorf("MidField: Expected 10, ok: %v, val: %v", okMid, fieldMid)
	}

	// Deeply nested anonymous field
	fieldDeep, okDeep := innerFieldValue.FindFieldFromTag("tag", "deep")
	if !okDeep || !fieldDeep.IsValid() || fieldDeep.String() != "deep_val" {
		t.Errorf("DeepField: Expected 'deep_val', ok: %v, val: %v", okDeep, fieldDeep)
	}

	// Field through a pointer field
	outerPtrValue, okOuterPtr := val.FindFieldFromTag("tag", "outer_ptr")
	if !okOuterPtr || !outerPtrValue.IsValid() {
		t.Fatalf("Could not find 'outer_ptr'")
	}
	// Now search within the struct pointed to by outerPtrValue
	fieldMidViaPtr, okMidViaPtr := outerPtrValue.FindFieldFromTag("tag", "mid")
	if !okMidViaPtr || !fieldMidViaPtr.IsValid() || fieldMidViaPtr.Int() != 10 {
		t.Errorf("MidField via OuterPtr: Expected 10, ok: %v, val: %v", okMidViaPtr, fieldMidViaPtr)
	}

	// Searching for a tag on a pointer field that itself points to a struct with the tag
	midPtrValue, okMidPtr := innerFieldValue.FindFieldFromTag("tag", "mid_ptr")
	if !okMidPtr || !midPtrValue.IsValid() {
		t.Fatalf("Could not find 'mid_ptr'")
	}
	deepViaMidPtr, okDeepViaMidPtr := midPtrValue.FindFieldFromTag("tag", "deep")
	if !okDeepViaMidPtr || !deepViaMidPtr.IsValid() || deepViaMidPtr.String() != "deep_val" {
		t.Errorf("DeepField via MidPtr: Expected 'deep_val', ok: %v, val: %v", okDeepViaMidPtr, deepViaMidPtr)
	}

	// Non-existent tag
	_, okNotFound := val.FindFieldFromTag("tag", "non_existent")
	if okNotFound {
		t.Error("Expected 'non_existent' tag to not be found")
	}
}

func TestIsNilable_value_test(t *testing.T) {
	var s string
	var ps *string
	var i int
	var pi *int
	var m map[string]int
	var pm *map[string]int
	var sl []int
	var psl *[]int
	var ch chan int
	var pch *chan int
	var fn func()
	var pfn *func()
	var iface any
	var piface *any
	var st struct{}
	var pst *struct{}

	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{"string", s, false},
		{"*string (nil)", ps, true},
		{"*string (non-nil)", &s, true},
		{"int", i, false},
		{"*int (nil)", pi, true},
		{"*int (non-nil)", &i, true},
		{"map (nil)", m, true},
		{"map (non-nil)", make(map[string]int), true},
		{"*map (nil)", pm, true},
		{"*map (non-nil)", &m, true},
		{"slice (nil)", sl, true},
		{"slice (non-nil)", make([]int, 0), true},
		{"*slice (nil)", psl, true},
		{"*slice (non-nil)", &sl, true},
		{"chan (nil)", ch, true},
		{"chan (non-nil)", make(chan int), true},
		{"*chan (nil)", pch, true},
		{"*chan (non-nil)", &ch, true},
		{"func (nil)", fn, true},
		{"func (non-nil)", func() {}, true},
		{"*func (nil)", pfn, true},
		{"*func (non-nil)", &fn, true},
		{"interface (nil)", iface, true},
		// For a non-nil interface holding a concrete value, reflect.ValueOf() returns a Value of the concrete kind.
		// IsNilable checks the Kind. String kind is not nilable. Pointer kind is.
		{"interface (non-nil, string)", any("hello"), false}, // reflect.ValueOf(any("hello")).Kind() is String. IsNilable(String) is false.
		{"interface (non-nil, *int)", any(&i), true},         // reflect.ValueOf(any(&i)).Kind() is Ptr. IsNilable(Ptr) is true.
		{"*interface (nil)", piface, true},
		{"*interface (non-nil)", &iface, true},
		{"struct", st, false},
		{"*struct (nil)", pst, true},
		{"*struct (non-nil)", &st, true},
		{"reflect.Value (invalid)", reflect.Value{}, true},
		{"reflect.Value (zero string)", reflect.ValueOf(""), false}, // String itself is not nilable
		{"map (nil from var)", (map[string]int)(nil), true},         // Corrected: pass the nil map directly
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(tt.value)
			// For the specific "reflect.Value (invalid)" case
			if tt.name == "reflect.Value (invalid)" {
				val = reflect.Value{} // Ensure it's an actual zero reflect.Value
			}
			if got := IsNilable(val); got != tt.want {
				t.Errorf("IsNilable(%v of type %T) = %v, want %v", tt.value, tt.value, got, tt.want)
			}
		})
	}
}

func TestUnwrap_Global_value_test(t *testing.T) {
	s := "test"
	ps := &s
	pps := &ps
	var nilPtr *string
	var nilInterface any = nilPtr
	var validInterfaceWithValue any = s
	var ptrToInterface any = &validInterfaceWithValue

	tests := []struct {
		name     string
		input    any
		expected any // For non-nil, expected interface value. For nil, expected kind.
		isNil    bool
		expKind  reflect.Kind // Expected Kind for nil values or specific checks
	}{
		{"string", s, s, false, reflect.String},
		{"*string", ps, s, false, reflect.String},
		{"**string", pps, s, false, reflect.String},
		{"nil *string", nilPtr, nil, true, reflect.Pointer},
		{"nil interface (from nil ptr)", nilInterface, nil, true, reflect.Pointer}, // Unwrap stops at nil pointer inside interface
		{"valid interface (string)", validInterfaceWithValue, s, false, reflect.String},
		{"*interface (to string)", ptrToInterface, s, false, reflect.String},
		{"reflect.Value of string", reflect.ValueOf(s), s, false, reflect.String},
		{"reflect.Value of *string", reflect.ValueOf(ps), s, false, reflect.String},
		{"invalid reflect.Value", reflect.Value{}, nil, true, reflect.Invalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputVal reflect.Value
			if rv, ok := tt.input.(reflect.Value); ok {
				inputVal = rv
			} else {
				inputVal = reflect.ValueOf(tt.input)
			}

			unwrapped := Unwrap(inputVal)

			// 1. Determine if the unwrapped value is actually nil (or equivalent, like Invalid)
			isActuallyNil := false
			if !unwrapped.IsValid() {
				isActuallyNil = true
			} else {
				switch unwrapped.Kind() {
				case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
					isActuallyNil = unwrapped.IsNil()
				default: // Other kinds (string, int, struct, etc.) cannot be nil.
					isActuallyNil = false
				}
			}

			// 2. Compare actual nil status with expected nil status
			if isActuallyNil != tt.isNil {
				t.Errorf("Unwrap(%s): Mismatch in nil status. Got isNil=%v, want isNil=%v. Value: '%v', Kind: %s",
					tt.name, isActuallyNil, tt.isNil, unwrapped, unwrapped.Kind())
			}

			// 3. If expecting non-nil, check value and kind
			if !tt.isNil {
				if !unwrapped.IsValid() { // Should have been caught by isActuallyNil check if tt.isNil was false
					t.Errorf("Unwrap(%s): Expected non-nil value, but got invalid. Expected: '%v' (Kind %s)",
						tt.name, tt.expected, tt.expKind)
				} else if isActuallyNil { // Should also have been caught by the previous check
					t.Errorf("Unwrap(%s): Expected non-nil value, but got programmatically nil. Expected: '%v' (Kind %s)",
						tt.name, tt.expected, tt.expKind)
				} else {
					// Compare actual value
					canCompareInterface := true
					switch unwrapped.Kind() {
					// For functions, direct comparison of unwrapped.Interface() might not be meaningful or stable.
					// For channels, direct comparison is also not typical for 'equality' of function.
					case reflect.Func, reflect.Chan:
						canCompareInterface = false
					}

					if canCompareInterface {
						// tt.expected might be nil for cases like a nil pointer within an interface that gets unwrapped.
						// However, if !tt.isNil, tt.expected should generally not be nil unless it's a specific test for zero values.
						// For most non-nil cases, unwrapped.Interface() should not panic.
						if unwrapped.Interface() != tt.expected {
							t.Errorf("Unwrap(%s): Value mismatch. Got '%v' (type %T), want '%v' (type %T). Kind: %s",
								tt.name, unwrapped.Interface(), unwrapped.Interface(), tt.expected, tt.expected, unwrapped.Kind())
						}
					} else if unwrapped.Kind() == reflect.Func && tt.expected != nil && !unwrapped.IsNil() {
						// Special handling if we expected a non-nil func, just check it's non-nil
						// tt.expected for func might just be a non-nil marker if direct comparison is hard
						if unwrapped.IsNil() {
							t.Errorf("Unwrap(%s): Expected a non-nil func, but got nil. Kind: %s", tt.name, unwrapped.Kind())
						}
					}

					if unwrapped.Kind() != tt.expKind {
						t.Errorf("Unwrap(%s): Kind mismatch. Got %s, want %s. Value: '%v'",
							tt.name, unwrapped.Kind(), tt.expKind, unwrapped)
					}
				}
			} else { // 4. If expecting nil (tt.isNil is true)
				// isActuallyNil should be true here due to the check at step 2.
				// So, we primarily care if the kind of this "nil" matches what we expect (e.g., nil Ptr vs Invalid).
				if unwrapped.Kind() != tt.expKind {
					t.Errorf("Unwrap(%s): Expected nil with kind %s, but got kind %s. Value: '%v'",
						tt.name, tt.expKind, unwrapped.Kind(), unwrapped)
				}
			}
		})
	}
}
