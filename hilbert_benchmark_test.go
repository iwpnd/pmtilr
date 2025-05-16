package pmtilr

import (
	"testing"
)

var (
	benchZ uint64 = 18
	benchX uint64 = 51542
	benchY uint64 = 92954
)

func BenchmarkZXYToHilbertTileID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = ZXYToHilbertTileID(benchZ, benchX, benchY)
	}
}

func BenchmarkFastZXYToHilbertTileID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = FastZXYToHilbertTileID(benchZ, benchX, benchY)
	}
}

func BenchmarkZXYFromHilbertTileID(b *testing.B) {
	// Precompute a valid tileID for decode benchmark
	tileID, _ := ZXYToHilbertTileID(benchZ, benchX, benchY)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = ZXYFromHilbertTileID(tileID)
	}
}

func BenchmarkFastZXYfromHilbertTileID(b *testing.B) {
	tileID, _ := FastZXYToHilbertTileID(benchZ, benchX, benchY)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = FastZXYfromHilbertTileID(tileID)
	}
}
