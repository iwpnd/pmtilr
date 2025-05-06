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
func rotate(n, x, y, rx, ry int) [2]int {
	if ry == 0 {
		if rx != 0 {
			return [2]int{n - 1 - y, n - 1 - x}
		}
		return [2]int{y, x}
	}

	return [2]int{x, y}
}

func ZxyToHilbertTileID(z, x, y uint64) (uint64, error) {
	if z > MaxZ {
		return 0, fmt.Errorf("zoom %d exceeds limit of %d", z, MaxZ)
	}

	if (x >= 1<<z) || y >= 1<<z {
		return 0, fmt.Errorf("tile coordinates x/y (%d/%d) outside of bounds for zoom %d", x, y, z)
	}

	// start at z 1
	accumulator := ((1<<z)*(1<<z) - 1) / 3
	pos := int(z - 1)

	// mutable copies of x,y
	tx, ty := int(x), int(y)

	for s := 1 << pos; s > 0; s >>= 1 {
		rx := tx & s
		ry := ty & s
		accumulator += ((3 * rx) ^ ry) * (1 << pos)
		rotatedXY := rotate(s, tx, ty, rx, ry)
		tx, ty = rotatedXY[0], rotatedXY[1]
		pos -= 1
	}

	return uint64(accumulator), nil
}

func ZxyFromHilbertTileID(i uint64) ([3]int, error) {
	z := ZoomFromHilbertTileID(i)
	if z > MaxZ {
		return [3]int{}, fmt.Errorf("tile zoom level %d exceeds maximum zoom level %d", z, MaxZ)
	}
	accumulator := ((1<<z)*(1<<z) - 1) / 3
	t := i - uint64(accumulator)
	x := 0
	y := 0
	n := 1 << z

	for s := 1; s < n; s <<= 1 {
		rx := s & int(t/2)
		ry := s & (int(t) ^ rx)
		rotatedXY := rotate(s, x, y, rx, ry)
		x, y = rotatedXY[0], rotatedXY[1]
		t = t / 2
		x += rx
		y += ry
	}

	return [3]int{z, x, y}, nil
}

func ZoomFromHilbertTileID(i uint64) int {
	c := 3*i + 1
	return int((bits.Len64(c) - 1) / 2)
}
