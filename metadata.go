package pmtilr

import (
	"context"
	"encoding/json"
	"fmt"
)

type VectorLayer struct {
	ID          string         `json:"id"`
	Fields      map[string]any `json:"fields"`
	Description string         `json:"description,omitempty"`
	MinZoom     int            `json:"minzoom,omitempty"`
	MaxZoom     int            `json:"maxzoom,omitempty"`
}

type Metadata struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Attribution  string        `json:"attribution"`
	Type         string        `json:"type"`
	Version      string        `json:"version"`
	VectorLayers []VectorLayer `json:"vector_layers"`

	metadataStr string // cache string representation
}

func (m *Metadata) ReadFrom(
	ctx context.Context,
	header HeaderV3,
	r RangeReader,
	decompress DecompressFunc,
) error {
	compressed, err := r.ReadRange(
		ctx,
		NewRange(header.MetadataOffset, header.MetadataLength),
	)
	if err != nil {
		return fmt.Errorf("reading metadata range: %w", err)
	}

	decompressed, err := decompress(compressed, header.InternalCompression)
	if err != nil {
		return fmt.Errorf("decompressing metadata: %w", err)
	}

	if err := json.Unmarshal(decompressed, m); err != nil {
		return fmt.Errorf("unmarshalling metadata: %w", err)
	}

	return nil
}

func (m Metadata) String() string {
	if m.metadataStr != "" {
		return m.metadataStr
	}

	jsonBytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return `{"error": "failed to marshal Metadata"}`
	}

	m.metadataStr = string(jsonBytes)

	return m.metadataStr
}
