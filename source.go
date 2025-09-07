package pmtilr

import (
	"context"
	"fmt"
	"sync"
)

// keyBufPool provides a shared buffer pool with 64-byte pre-allocated buffers
var keyBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 64) // Pre-allocate 64 bytes capacity, sufficient for key pattern
		return &buf
	},
}

// SourceConfig holds customization options for a Source.
type SourceConfig struct {
	// decompress is the function used to unpack raw tile data.
	decompress DecompressFunc
}

// SourceConfigOption is a functional option for configuring a Source.
type SourceConfigOption = func(config *SourceConfig)

// WithCustomDecompressFunc sets a custom decompression function on the SourceConfig.
func WithCustomDecompressFunc(decompressFn DecompressFunc) SourceConfigOption {
	return func(config *SourceConfig) {
		config.decompress = decompressFn
	}
}

// Source provides read access to protomap tiles, supporting concurrent
// loads with singleflight deduplication.
type Source struct {
	reader     RangeReader   // Underlying reader for HTTP range requests
	header     *HeaderV3     // Parsed header containing tile layout and ETag
	meta       *Metadata     // Metadata for tile index and offsets
	config     *SourceConfig // Configuration options
	repository *Repository   // Repository for actual tile reads
}

// NewSource initializes a Source, optionally applying SourceConfigOptions,
// and immediately loads the header and metadata.
// It returns an error if initial header or metadata reading fails.
func NewSource(ctx context.Context, uri string, options ...SourceConfigOption) (*Source, error) {
	config := &SourceConfig{
		decompress: Decompress,
	}
	// Apply user options
	for _, o := range options {
		o(config)
	}

	reader, err := NewRangeReader(ctx, uri)
	if err != nil {
		return nil, err
	}

	// Create Source with defaults
	s := &Source{
		reader: reader,
		header: &HeaderV3{},
		meta:   &Metadata{},
		config: config,
	}

	if err := s.header.ReadFrom(ctx, s.reader); err != nil {
		return nil, err
	}

	if err := s.meta.ReadFrom(ctx, *s.header, s.reader, s.config.decompress); err != nil {
		return nil, err
	}

	// Initialize repository for tile decoding
	repo, err := NewRepository()
	if err != nil {
		return nil, err
	}
	s.repository = repo

	return s, nil
}

// Tile returns the raw tile bytes for the specified z, x, y. If the zoom
// level is within the configured singleFlightZoomRange, concurrent calls
// for the same tile are collapsed into a single request.
func (s *Source) Tile(ctx context.Context, z, x, y uint64) ([]byte, error) {
	if z < uint64(s.header.MinZoom) || z > uint64(s.header.MaxZoom) {
		return []byte{}, fmt.Errorf(
			"invalid zoom: %d for allowed range of %d to %d",
			z,
			s.header.MinZoom,
			s.header.MaxZoom,
		)
	}
	// if s.useSingleFlight(z) {
	// 	// Deduplicate concurrent loads of the same tile
	// 	key := buildSingleflightKey(s.header.Etag, z, x, y)
	// 	data, err, _ := s.sg.Do(key, func() ([]byte, error) {
	// 		return s.repository.Tile(ctx, s.Header(), s.reader, s.config.decompress, z, x, y)
	// 	})
	// 	if err != nil {
	// 		return nil, fmt.Errorf("reading tile with singleflight: %w", err)
	// 	}
	// 	return data, nil
	// }
	// No deduplication: direct repository call
	return s.repository.Tile(ctx, s.Header(), s.reader, s.config.decompress, z, x, y)
}

// Header returns a copy of the current header.
func (s *Source) Header() HeaderV3 {
	return *s.header
}

// Meta returns a copy of the current metadata.
func (s *Source) Meta() Metadata {
	return *s.meta
}

// Close the source and its dependencies.
func (s *Source) Close() {
	s.repository.Close()
}
