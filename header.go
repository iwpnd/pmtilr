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

type HeaderV3 struct {
	Etag                string      `json:"etag"`
	SpecVersion         uint8       `json:"spec_version"`
	RootOffset          uint64      `json:"root_offset"`
	RootLength          uint64      `json:"root_length"`
	MetadataOffset      uint64      `json:"metadata_offset"`
	MetadataLength      uint64      `json:"metadata_length"`
	LeafDirectoryOffset uint64      `json:"leaf_directory_offset"`
	LeafDirectoryLength uint64      `json:"leaf_directory_length"`
	TileDataOffset      uint64      `json:"tile_data_offset"`
	TileDataLength      uint64      `json:"tile_data_length"`
	AddressedTilesCount uint64      `json:"addressed_tiles_count"`
	TileEntriesCount    uint64      `json:"tile_entries_count"`
	TileContentsCount   uint64      `json:"tile_contents_count"`
	Clustered           bool        `json:"clustered"`
	InternalCompression Compression `json:"internal_compression"`
	TileCompression     Compression `json:"tile_compression"`
	TileType            TileType    `json:"tile_type"`
	MinZoom             uint8       `json:"min_zoom"`
	MaxZoom             uint8       `json:"max_zoom"`
	MinLonE7            int32       `json:"min_lon_e7"`
	MinLatE7            int32       `json:"min_lat_e7"`
	MaxLonE7            int32       `json:"max_lon_e7"`
	MaxLatE7            int32       `json:"max_lat_e7"`
	CenterZoom          uint8       `json:"center_zoom"`
	CenterLonE7         int32       `json:"center_lon_e7"`
	CenterLatE7         int32       `json:"center_lat_e7"`
}

func NewHeader(r io.Reader) (*HeaderV3, error) {
	h := &HeaderV3{}
	d := make([]byte, HeaderSizeBytes)
	_, err := io.ReadFull(r, d)
	if err != nil {
		return h, fmt.Errorf("reading header: %w", err)
	}
	if err := h.deserialize(d); err != nil {
		return h, err
	}
	return h, nil
}

func (h *HeaderV3) ReadFrom(ctx context.Context, r RangeReader) (err error) {
	b, err := r.ReadRange(
		ctx,
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
		return fmt.Errorf("unsupported spec version: %w", err)
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
	h.MinLonE7 = int32(binary.LittleEndian.Uint32(d[102:106])) //nolint:gosec
	h.MinLatE7 = int32(binary.LittleEndian.Uint32(d[106:110])) //nolint:gosec
	h.MaxLonE7 = int32(binary.LittleEndian.Uint32(d[110:114])) //nolint:gosec
	h.MaxLatE7 = int32(binary.LittleEndian.Uint32(d[114:118])) //nolint:gosec

	// 6) center point
	h.CenterZoom = d[118]
	h.CenterLonE7 = int32(binary.LittleEndian.Uint32(d[119:123])) //nolint:gosec
	h.CenterLatE7 = int32(binary.LittleEndian.Uint32(d[123:127])) //nolint:gosec

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
