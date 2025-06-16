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
	z := uint64(zCalc) //nolint:gosec
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

// based on a discussion in PMTiles/Issue#383
// https://github.com/protomaps/PMTiles/issues/393
const invalidTileID uint64 = 0x5555555555555555

// FastZXYToHilbertTileID converts tile coordinates (z, x, y) to a compact 64-bit ID.
func FastZXYToHilbertTileID(z, x, y uint64) (uint64, error) {
	if z > 31 {
		return 0, errors.New("tile zoom exceeds 64-bit limit")
	}
	if x >= 1<<z || y >= 1<<z {
		return 0, errors.New("tile x/y outside zoom level bounds")
	}

	// prefix is ((1 << (2*z)) - 1) / 3
	prefix := ((uint64(1) << (2 * z)) - 1) / 3

	var state, result uint64
	const lut1 = 0x361E9CB4
	const lut2 = 0x8FE65831

	// Iterate bits from highest zoom down to 0
	for i := z; i > 0; i-- {
		shift := i - 1
		// build row index: 3 bits = [state(2)] [x_i(1)] [y_i(1)]
		row2 := (state << 3) | ((x>>shift)&1)<<2 | ((y>>shift)&1)<<1
		result = (result << 2) | ((lut1 >> row2) & 3)
		state = (lut2 >> row2) & 3
	}

	return prefix + result, nil
}

// FastZXYfromHilbertTileID converts a 64-bit tile ID back into (z, x, y) coordinates.
func FastZXYfromHilbertTileID(tileID uint64) ([3]uint64, error) {
	if tileID >= invalidTileID {
		return [3]uint64{}, errors.New("tile zoom exceeds 64-bit limit")
	}

	// Determine zoom level z by finding largest z such that (1 << (2*(z+1))) <= 3*tileID+1
	var z uint64
	for (uint64(1) << (2 * (z + 1))) <= 3*tileID+1 {
		z++
	}

	// subtract prefix
	prefix := ((uint64(1) << (2 * z)) - 1) / 3
	code := tileID - prefix

	var state uint64
	const lutX = 0x936C
	const lutY = 0x39C6
	const lutState = 0x3E6B94C1

	var x, y uint64
	// iterate over code bits in pairs
	for i := 2 * z; i > 0; i -= 2 {
		shift := i - 2
		codeBits := (code >> shift) & 3
		row := (state << 2) | codeBits
		x = (x << 1) | ((lutX >> row) & 1)
		y = (y << 1) | ((lutY >> row) & 1)
		state = (lutState >> (2 * row)) & 3
	}

	return [3]uint64{z, x, y}, nil
}
