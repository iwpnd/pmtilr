package pmtilr

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func makeValidHeaderBytes(modifier func([]byte) []byte) []byte {
	data := make([]byte, HeaderSizeBytes)

	copy(data[0:7], []byte("PMTiles"))              // magic
	data[7] = 3                                     // version
	binary.LittleEndian.PutUint64(data[8:16], 1000) // RootOffset
	// other fields are 0d

	// apply custom changes based on test
	if modifier != nil {
		data = modifier(data)
	}

	return data
}

func TestNewHeader(t *testing.T) {
	tests := []struct {
		name     string
		modify   func([]byte) []byte
		wantErr  bool
		wantSpec uint8
	}{
		{
			name:     "valid header",
			modify:   nil,
			wantErr:  false,
			wantSpec: 3,
		},
		{
			name: "invalid magic",
			modify: func(data []byte) []byte {
				copy(data[0:7], []byte("Invalid"))
				return data
			},
			wantErr: true,
		},
		{
			name: "unsupported version",
			modify: func(data []byte) []byte {
				data[7] = 1
				return data
			},
			wantErr: true,
		},
		{
			name: "incomplete data",
			modify: func(data []byte) []byte {
				data = data[:10] // truncated
				return data
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := makeValidHeaderBytes(tc.modify)
			r := bytes.NewReader(data)
			h, err := NewHeader(r)

			if (err != nil) != tc.wantErr {
				t.Errorf("expected error: %v, got: %v", tc.wantErr, err)
			}

			if err == nil && h.SpecVersion != tc.wantSpec {
				t.Errorf("expected spec version %d, got %d", tc.wantSpec, h.SpecVersion)
			}
		})
	}
}

func TestHeaderString(t *testing.T) {
	h := HeaderV3{
		SpecVersion:         3,
		RootOffset:          1234,
		TileCompression:     CompressionGZIP,
		TileType:            TileTypeMVT,
		InternalCompression: CompressionNone,
		Clustered:           true,
		MinZoom:             2,
		MaxZoom:             12,
	}

	out := h.String()
	if !strings.Contains(out, `"SpecVersion": 3`) {
		t.Errorf("expected SpecVersion in JSON, got %s", out)
	}
	if !strings.Contains(out, `"gzip"`) {
		t.Errorf("expected Compression to be marshaled as string, got %s", out)
	}
	if !strings.Contains(out, `"mvt"`) {
		t.Errorf("expected TileType to be marshaled as string, got %s", out)
	}
}
