package pmtilr

import (
	"testing"
)

func TestFastMatchesOriginal(t *testing.T) {
	t.Parallel()

	inputs := [][3]uint64{
		{3, 1, 3},
		{5, 7, 12},
		{10, 205, 342},
	}

	for _, in := range inputs {
		z, x, y := in[0], in[1], in[2]

		origID, err := ZXYToHilbertTileID(z, x, y)
		if err != nil {
			t.Errorf("original ZXYToHilbertTileID(%d, %d, %d) returned error: %v", z, x, y, err)
			continue
		}
		fastID, err := FastZXYToHilbertTileID(z, x, y)
		if err != nil {
			t.Errorf("FastZXYToHilbertTileID(%d, %d, %d) returned error: %v", z, x, y, err)
			continue
		}
		if origID != fastID {
			t.Errorf("encode mismatch for (%d, %d, %d): original=%d fast=%d", z, x, y, origID, fastID)
		}

		// Test decoding
		origOut, err := ZXYFromHilbertTileID(origID)
		if err != nil {
			t.Errorf("original ZXYFromHilbertTileID(%d) returned error: %v", origID, err)
			continue
		}

		fastOut, err := FastZXYfromHilbertTileID(fastID)
		if err != nil {
			t.Errorf("fast FastZXYFromHilbertTileID(%d) returned error: %v", fastID, err)
			continue
		}

		if origOut != fastOut {
			t.Errorf("decode mismatch for ID %d: original=%v fast=%v", origID, origOut, fastOut)
			continue
		}

		if fastOut != in {
			t.Errorf("decode mismatch for ID %d: input=%v fast=%v", origID, in, fastOut)
			continue
		}
		if origOut != in {
			t.Errorf("decode mismatch for ID %d: input=%v original=%v", origID, in, origOut)
			continue
		}
	}
}
