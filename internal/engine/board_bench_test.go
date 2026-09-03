package engine

import "testing"

func BenchmarkOccupancy(b *testing.B) {
	p, _ := Prepare(fixture())
	k := p.Level.InitialKey()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Occupancy(p, k)
	}
}
func BenchmarkGenerateMoves(b *testing.B) {
	p, _ := Prepare(fixture())
	k := p.Level.InitialKey()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateMoves(p, k, nil)
	}
}
