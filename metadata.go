package pmtilr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

type Metadata struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Attribution  string `json:"attribution"`
	Type         string `json:"type"`
	Version      string `json:"version"`
	VectorLayers []any  `json:"vector_layers"`
}

func (m *Metadata) ReadFrom(
	ctx context.Context,
	header HeaderV3,
	r RangeReader,
	decompress DecompressFunc,
) error {
	data, err := r.ReadRange(
		ctx,
		NewRange(header.MetadataOffset, header.MetadataLength),
	)
	if err != nil {
		return fmt.Errorf("reading metadata range: %w", err)
	}

	decompReader, err := decompress(bytes.NewReader(data), header.InternalCompression)
	if err != nil {
		return fmt.Errorf("decompressing metadata: %w", err)
	}

	jsonData, err := io.ReadAll(decompReader)
	if err != nil {
		return fmt.Errorf("reading decompressed metadata: %w", err)
	}

	if closer, ok := decompReader.(io.Closer); ok {
		cerr := closer.Close()
		if cerr != nil {
			return fmt.Errorf("closing decompression reader: %w", cerr)
		}
	}

	if err := json.Unmarshal(jsonData, m); err != nil {
		return fmt.Errorf("unmarshalling metadata: %w", err)
	}

	return nil
}

func (m Metadata) String() string {
	jsonBytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return `{"error": "failed to marshal Metadata"}`
	}
	return string(jsonBytes)
}
