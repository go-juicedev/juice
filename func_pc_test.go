package juice

import (
	"reflect"
	"runtime"
	"sync"
	"testing"
)

type testStruct struct{}

func (t testStruct) testMethod() {}

func (t *testStruct) pointerMethod() {}

func helperFunc1() {}

func BenchmarkFuncName(b *testing.B) {
	t := testStruct{}
	addr := runtime.FuncForPC(reflect.ValueOf(t.testMethod).Pointer()).Entry()

	b.Run("without_cache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = runtimeFuncName(addr)
		}
	})

	b.Run("with_cache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cachedRuntimeFuncName(addr)
		}
	})

	b.Run("multiple_funcs_without_cache", func(b *testing.B) {
		funcs := []uintptr{
			runtime.FuncForPC(reflect.ValueOf(testStruct{}.testMethod).Pointer()).Entry(),
			runtime.FuncForPC(reflect.ValueOf(runtimeFuncName).Pointer()).Entry(),
			runtime.FuncForPC(reflect.ValueOf(cachedRuntimeFuncName).Pointer()).Entry(),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = runtimeFuncName(funcs[i%len(funcs)])
		}
	})

	b.Run("multiple_funcs_with_cache", func(b *testing.B) {
		funcs := []uintptr{
			runtime.FuncForPC(reflect.ValueOf(testStruct{}.testMethod).Pointer()).Entry(),
			runtime.FuncForPC(reflect.ValueOf(runtimeFuncName).Pointer()).Entry(),
			runtime.FuncForPC(reflect.ValueOf(cachedRuntimeFuncName).Pointer()).Entry(),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cachedRuntimeFuncName(funcs[i%len(funcs)])
		}
	})

	b.Run("concurrent_without_cache", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = runtimeFuncName(addr)
			}
		})
	})

	b.Run("concurrent_with_cache", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = cachedRuntimeFuncName(addr)
			}
		})
	})
}

func BenchmarkFuncNameAlloc(b *testing.B) {
	t := testStruct{}
	addr := runtime.FuncForPC(reflect.ValueOf(t.testMethod).Pointer()).Entry()

	b.Run("alloc_without_cache", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = runtimeFuncName(addr)
		}
	})

	b.Run("alloc_with_cache", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = cachedRuntimeFuncName(addr)
		}
	})

	b.Run("receiver_types", func(b *testing.B) {
		t := &testStruct{}
		addrs := []uintptr{
			runtime.FuncForPC(reflect.ValueOf(t.testMethod).Pointer()).Entry(),
			runtime.FuncForPC(reflect.ValueOf(t.pointerMethod).Pointer()).Entry(),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cachedRuntimeFuncName(addrs[i%2])
		}
	})

	b.Run("many_different_funcs", func(b *testing.B) {
		var funcs []uintptr
		for i := 0; i < 1000; i++ {
			f := func() {}
			funcs = append(funcs, runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Entry())
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cachedRuntimeFuncName(funcs[i%len(funcs)])
		}
	})

	b.Run("concurrent_write", func(b *testing.B) {
		var funcs sync.Map
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			f := func() {}
			addr := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Entry()
			for pb.Next() {
				_ = cachedRuntimeFuncName(addr)
				funcs.Store(addr, struct{}{})
			}
		})
	})

	b.Run("different_path_lengths", func(b *testing.B) {
		longNameFunc := func() {}
		addrs := []uintptr{
			runtime.FuncForPC(reflect.ValueOf(helperFunc1).Pointer()).Entry(),
			runtime.FuncForPC(reflect.ValueOf(longNameFunc).Pointer()).Entry(),
			runtime.FuncForPC(reflect.ValueOf(testStruct{}.testMethod).Pointer()).Entry(),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cachedRuntimeFuncName(addrs[i%len(addrs)])
		}
	})

	b.Run("cache_hit_ratio", func(b *testing.B) {
		funcs := make([]uintptr, 100)
		for i := range funcs {
			f := func() {}
			funcs[i] = runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Entry()
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if i%100 == 0 {
				f := func() {}
				_ = cachedRuntimeFuncName(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Entry())
			} else {
				_ = cachedRuntimeFuncName(funcs[i%len(funcs)])
			}
		}
	})

	b.Run("reflection_overhead", func(b *testing.B) {
		t := testStruct{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			v := reflect.ValueOf(t.testMethod)
			_ = cachedRuntimeFuncName(runtime.FuncForPC(v.Pointer()).Entry())
		}
	})
}

func BenchmarkFuncNameUnderMemoryPressure(b *testing.B) {
	pressure := make([][]byte, 0)
	for i := 0; i < 100; i++ {
		pressure = append(pressure, make([]byte, 1024*1024)) // 1MB each
	}

	b.Run("under_memory_pressure", func(b *testing.B) {
		t := testStruct{}
		addr := runtime.FuncForPC(reflect.ValueOf(t.testMethod).Pointer()).Entry()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = cachedRuntimeFuncName(addr)
			if i%100 == 0 {
				pressure = append(pressure, make([]byte, 1024*1024))
			}
		}
	})
}
