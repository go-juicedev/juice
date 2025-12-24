package sql

import (
	"reflect"
	"testing"
)

var sinkBench any

type BenchStruct struct {
	A int
	B int
}

func BenchmarkTypeAssertUnbox(b *testing.B) {
	s := BenchStruct{A: 1, B: 2}
	v := reflect.ValueOf(s)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		val, _ := reflect.TypeAssert[BenchStruct](v)
		sinkBench = val
	}
}

func BenchmarkInterfaceUnbox(b *testing.B) {
	s := BenchStruct{A: 1, B: 2}
	v := reflect.ValueOf(s)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		val := v.Interface().(BenchStruct)
		sinkBench = val
	}
}
