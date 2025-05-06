package pmtilr

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/segmentio/ksuid"
)

const (
	HeaderOffset    = 0
	HeaderSizeBytes = 127
)

func NewHeader(r io.Reader) (*HeaderV3, error) {
	h := &HeaderV3{}
	d := make([]byte, HeaderSizeBytes)
	_, err := io.ReadFull(r, d)
	if err != nil {
		return h, fmt.Errorf("reading header: %v", err)
	}
	if err := h.deserialize(d); err != nil {
		return h, err
	}
	return h, nil
}

func (h *HeaderV3) ReadFrom(r RangeReader) (err error) {
	b, err := r.ReadRange(
		context.Background(),
		NewRange(HeaderOffset, HeaderSizeBytes),
	)
	if err != nil {
		return fmt.Errorf("reading header: %w", err)
	}
	newHeader, err := NewHeader(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("reading header: %w", err)
	}
	if newHeader.Etag == "" {
		newHeader.Etag = ksuid.New().String()
	}

	*h = *newHeader

	return
}

func (h HeaderV3) String() string {
	jsonBytes, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return `{"error": "failed to marshal HeaderV3"}`
	}
	return string(jsonBytes)
}

func (h *HeaderV3) deserialize(d []byte) error {
	// 1) magic
	if string(d[0:7]) != "PMTiles" {
		return fmt.Errorf("magic number not detected; confirm this is a PMTiles archive")
	}

	// 2) version
	ver, err := h.version(d[7])
	if err != nil {
		return fmt.Errorf("unsupported spec version: %v", err)
	}
	h.SpecVersion = ver

	// 3) big‑grained fields
	h.RootOffset = binary.LittleEndian.Uint64(d[8:16])
	h.RootLength = binary.LittleEndian.Uint64(d[16:24])
	h.MetadataOffset = binary.LittleEndian.Uint64(d[24:32])
	h.MetadataLength = binary.LittleEndian.Uint64(d[32:40])
	h.LeafDirectoryOffset = binary.LittleEndian.Uint64(d[40:48])
	h.LeafDirectoryLength = binary.LittleEndian.Uint64(d[48:56])
	h.TileDataOffset = binary.LittleEndian.Uint64(d[56:64])
	h.TileDataLength = binary.LittleEndian.Uint64(d[64:72])
	h.AddressedTilesCount = binary.LittleEndian.Uint64(d[72:80])
	h.TileEntriesCount = binary.LittleEndian.Uint64(d[80:88])
	h.TileContentsCount = binary.LittleEndian.Uint64(d[88:96])

	// 4) flags & enums
	h.Clustered = (d[96] == 0x1)
	h.InternalCompression = Compression(d[97])
	h.TileCompression = Compression(d[98])
	h.TileType = TileType(d[99])

	// 5) zoom & bounds
	h.MinZoom = d[100]
	h.MaxZoom = d[101]
	h.MinLonE7 = int32(binary.LittleEndian.Uint32(d[102:106]))
	h.MinLatE7 = int32(binary.LittleEndian.Uint32(d[106:110]))
	h.MaxLonE7 = int32(binary.LittleEndian.Uint32(d[110:114]))
	h.MaxLatE7 = int32(binary.LittleEndian.Uint32(d[114:118]))

	// 6) center point
	h.CenterZoom = d[118]
	h.CenterLonE7 = int32(binary.LittleEndian.Uint32(d[119:123]))
	h.CenterLatE7 = int32(binary.LittleEndian.Uint32(d[123:127]))

	return nil
}

func (h *HeaderV3) version(d byte) (uint8, error) {
	switch d {
	case 1, 2:
		return 0, fmt.Errorf("spec version %d is unsupported", d)
	case 3:
		return 3, nil
	default:
		return 0, fmt.Errorf("unknown version")
	}
}

type HeaderV3 struct {
	Etag                string
	SpecVersion         uint8
	RootOffset          uint64
	RootLength          uint64
	MetadataOffset      uint64
	MetadataLength      uint64
	LeafDirectoryOffset uint64
	LeafDirectoryLength uint64
	TileDataOffset      uint64
	TileDataLength      uint64
	AddressedTilesCount uint64
	TileEntriesCount    uint64
	TileContentsCount   uint64
	Clustered           bool
	InternalCompression Compression
	TileCompression     Compression
	TileType            TileType
	MinZoom             uint8
	MaxZoom             uint8
	MinLonE7            int32
	MinLatE7            int32
	MaxLonE7            int32
	MaxLatE7            int32
	CenterZoom          uint8
	CenterLonE7         int32
	CenterLatE7         int32
}
