package pmtilr

import (
	"errors"
	"fmt"
	"math/bits"
)

const (
	MaxZ = 26
)

// rotate to layout points on the hilbert curve.
//
// ry=1 no change
// ry=0,rx=0 swap
// ry=0,rx=1 rotate 180degree
func rotate(n, x, y, rx, ry uint64) [2]uint64 {
	if ry == 0 {
		if rx != 0 {
			return [2]uint64{n - 1 - y, n - 1 - x}
		}
		return [2]uint64{y, x}
	}

	return [2]uint64{x, y}
}

func ZXYToHilbertTileID(z, x, y uint64) (uint64, error) {
	if z > MaxZ {
		return 0, fmt.Errorf("zoom %d exceeds limit of %d", z, MaxZ)
	}

	if (x >= 1<<z) || y >= 1<<z {
		return 0, fmt.Errorf("tile coordinates x/y (%d/%d) outside of bounds for zoom %d", x, y, z)
	}

	// start at z 1
	var accumulator uint64
	accumulator = ((1<<z)*(1<<z) - 1) / 3
	pos := z - 1

	// mutable copies of x,y
	tx, ty := x, y

	for s := uint64(1) << pos; s > 0; s >>= 1 {
		rx := tx & s
		ry := ty & s
		accumulator += ((3 * rx) ^ ry) * (1 << pos)
		rotatedXY := rotate(s, tx, ty, rx, ry)
		tx, ty = rotatedXY[0], rotatedXY[1]
		pos -= 1
	}

	return accumulator, nil
}

func ZXYFromHilbertTileID(i uint64) ([3]uint64, error) {
	// overflow check
	if i >= invalidTileID {
		return [3]uint64{}, errors.New("tile zoom exceeds 64-bit limit")
	}

	zCalc := ZoomFromHilbertTileID(i)
	z := uint64(zCalc)
	if z > MaxZ {
		return [3]uint64{}, fmt.Errorf("tile zoom level %d exceeds maximum %d", z, MaxZ)
	}

	// compute prefix = (1<<(2*z) - 1) / 3
	prefix := (uint64(1)<<(2*z) - 1) / 3
	t := i - prefix

	var x, y uint64
	// decode bits level by level
	for a := uint64(0); a < z; a++ {
		s := uint64(1) << a
		// extract bits: bit1->rx, bit0->ry (undo (3*rx)^ry encoding)
		rx := (t >> 1) & 1
		ry := (t & 1) ^ rx

		// undo rotation using existing rotate
		rot := rotate(s, x, y, rx, ry)
		x, y = rot[0], rot[1]

		// step into quadrant
		x += rx * s
		y += ry * s

		// consume two bits
		t >>= 2
	}

	return [3]uint64{z, x, y}, nil
}

func ZoomFromHilbertTileID(i uint64) int {
	c := 3*i + 1
	return (bits.Len64(c) - 1) / 2
}
