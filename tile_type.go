package pmtilr

import (
	"encoding/json"
	"fmt"
)

type TileType uint8

const (
	TileTypeUnknown = iota
	TileTypeMVT
	TileTypePNG
	TileTypeJPEG
	TileTypeWebp
	TileTypeAvif
	TileTypeMLT
)

var tileTypeOptions = map[TileType]string{
	TileTypeUnknown: "unknown",
	TileTypeMVT:     "mvt",
	TileTypeAvif:    "avif",
	TileTypeJPEG:    "jpeg",
	TileTypeWebp:    "webp",
	TileTypePNG:     "png",
	TileTypeMLT:     "mlt",
}

func (t TileType) String() string {
	return tileTypeOptions[t]
}

func (c TileType) MarshalJSON() ([]byte, error) {
	str, ok := tileTypeOptions[c]
	if !ok {
		str = tileTypeOptions[TileTypeUnknown]
	}
	return json.Marshal(str)
}

func (t TileType) Ext() string {
	return fmt.Sprintf(".%s", tileTypeOptions[t])
}

func (t TileType) ToContentType() (string, bool) {
	switch t {
	case TileTypeMVT:
		return "application/x-protobuf", true
	case TileTypeMLT:
		return "application/vnd.maplibre-vector-tile", true
	case TileTypeAvif:
		return "image/avif", true
	case TileTypeJPEG:
		return "image/jpeg", true
	case TileTypePNG:
		return "image/png", true
	case TileTypeWebp:
		return "image/webp", true
	default:
		return "", false
	}
}
