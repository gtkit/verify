package verify

import "testing"

func BenchmarkStructErr(b *testing.B) {
	if err := New(); err != nil {
		b.Fatalf("New() error = %v", err)
	}
	err := Struct(safetyPayload{Name: "a", Email: "bad"})
	if err == nil {
		b.Fatal("expected validation error")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if out := StructErr(err); out == nil {
			b.Fatal("expected translated error")
		}
	}
}

func BenchmarkFieldErr(b *testing.B) {
	if err := New(); err != nil {
		b.Fatalf("New() error = %v", err)
	}
	err := Field("not-a-number", "required,numeric")
	if err == nil {
		b.Fatal("expected validation error")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if out := FieldErr("age", err); out == nil {
			b.Fatal("expected translated error")
		}
	}
}
