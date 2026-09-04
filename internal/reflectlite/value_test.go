package reflectlite

import (
	"reflect"
	"testing"
)

func TestLookupFieldByTag_value_test(t *testing.T) {
	type Inner struct {
		DeepField string `tag:"deep"`
	}
	type Outer struct {
		Inner
		Direct string `tag:"direct"`
	}

	t.Run("direct field", func(t *testing.T) {
		v := reflect.ValueOf(Outer{DeepField: "deep_val", Direct: "hello"})
		got, ok := LookupFieldByTag(v, "tag", "direct")
		if !ok || got.String() != "hello" {
			t.Errorf("direct field: ok = %v, got = %v", ok, got)
		}
	})

	t.Run("anonymous embedded field", func(t *testing.T) {
		v := reflect.ValueOf(Outer{DeepField: "deep_val"})
		got, ok := LookupFieldByTag(v, "tag", "deep")
		if !ok || got.String() != "deep_val" {
			t.Errorf("anonymous embedded field: ok = %v, got = %v", ok, got)
		}
	})

	t.Run("pointer input", func(t *testing.T) {
		v := reflect.ValueOf(&Outer{Direct: "ptr_val"})
		got, ok := LookupFieldByTag(v, "tag", "direct")
		if !ok || got.String() != "ptr_val" {
			t.Errorf("pointer input: ok = %v, got = %v", ok, got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := LookupFieldByTag(reflect.ValueOf(Outer{}), "tag", "missing")
		if ok {
			t.Error("expected not found")
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		_, ok := LookupFieldByTag(reflect.Value{}, "tag", "direct")
		if ok {
			t.Error("expected not found for invalid value")
		}
	})

	t.Run("non-struct", func(t *testing.T) {
		_, ok := LookupFieldByTag(reflect.ValueOf(123), "tag", "direct")
		if ok {
			t.Error("expected not found for non-struct")
		}
	})
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
				case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
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
