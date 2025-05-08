package pmtilr

import (
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

func ZxyToHilbertTileID(z, x, y uint64) (uint64, error) {
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

func ZxyFromHilbertTileID(i uint64) ([3]uint64, error) {
	z := uint64(ZoomFromHilbertTileID(i)) //nolint:gosec
	if z > MaxZ {
		return [3]uint64{}, fmt.Errorf("tile zoom level %d exceeds maximum zoom level %d", z, MaxZ)
	}
	var accumulator, x, y, n uint64
	accumulator = ((1<<z)*(1<<z) - 1) / 3
	t := i - accumulator
	x = 0
	y = 0
	n = 1 << z

	for s := uint64(1); s < n; s <<= 1 {
		rx := s & t / 2
		ry := s & (t ^ rx)
		rotatedXY := rotate(s, x, y, rx, ry)
		x, y = rotatedXY[0], rotatedXY[1]
		t /= 2
		x += rx
		y += ry
	}

	return [3]uint64{z, x, y}, nil
}

func ZoomFromHilbertTileID(i uint64) int {
	c := 3*i + 1
	return (bits.Len64(c) - 1) / 2
}
