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
)

var tileTypeOptions = map[TileType]string{
	TileTypeUnknown: "unknown",
	TileTypeMVT:     "mvt",
	TileTypeAvif:    "avif",
	TileTypeJPEG:    "jpeg",
	TileTypeWebp:    "webp",
	TileTypePNG:     "png",
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
