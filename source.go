package pmtilr

import (
	"context"
	"fmt"

	"github.com/brunomvsouza/singleflight"
)

const (
	singleFlightKeyTemplate    = "%s:%d:%d:%d" // etag:z:x:y
	singleFlightDefaultMinZoom = 1
	singleFlightDefaultMaxZoom = 10
)

// SourceConfig holds customization options for a Source.
type SourceConfig struct {
	// decompress is the function used to unpack raw tile data.
	decompress DecompressFunc
	// singleFlightZoomRange defines the inclusive min/max zoom levels
	// at which singleflight de-duplication is active.
	singleFlightZoomRange ZoomRange
}

// SourceConfigOption is a functional option for configuring a Source.
type SourceConfigOption = func(config *SourceConfig)

// WithCustomDecompressFunc sets a custom decompression function on the SourceConfig.
func WithCustomDecompressFunc(decompressFn DecompressFunc) SourceConfigOption {
	return func(config *SourceConfig) {
		config.decompress = decompressFn
	}
}

// WithSingleFlightZoomRange sets the zoom range at which singleflight
// deduplication will be applied.
func WithSingleFlightZoomRange(zrange [2]uint64) SourceConfigOption {
	return func(config *SourceConfig) {
		config.singleFlightZoomRange = zrange
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

	// sg serializes concurrent Tile() calls for the same header.etag/z/x/y
	sg *singleflight.Group[string, []byte]
}

// NewSource initializes a Source, optionally applying SourceConfigOptions,
// and immediately loads the header and metadata.
// It returns an error if initial header or metadata reading fails.
func NewSource(reader RangeReader, options ...SourceConfigOption) (*Source, error) {
	config := &SourceConfig{
		decompress:            Decompress,
		singleFlightZoomRange: NewZoomRange(singleFlightDefaultMinZoom, singleFlightDefaultMaxZoom),
	}
	// Apply user options
	for _, o := range options {
		o(config)
	}
	// Validate zoom range
	if err := config.singleFlightZoomRange.Validate(); err != nil {
		return nil, fmt.Errorf("invalid single flight zoom range: %w", err)
	}

	// Create Source with defaults
	s := &Source{
		reader: reader,
		header: &HeaderV3{},
		meta:   &Metadata{},
		config: config,
		sg:     &singleflight.Group[string, []byte]{},
	}

	if err := s.header.ReadFrom(s.reader); err != nil {
		return nil, err
	}

	if err := s.meta.ReadFrom(*s.header, s.reader, s.config.decompress); err != nil {
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

// useSingleFlight reports whether singleflight deduplication is enabled
// for the given zoom level z.
func (s *Source) useSingleFlight(z uint64) bool {
	return z >= s.config.singleFlightZoomRange.MinZoom() &&
		z <= s.config.singleFlightZoomRange.MaxZoom()
}

// Tile returns the raw tile bytes for the specified z, x, y. If the zoom
// level is within the configured singleFlightZoomRange, concurrent calls
// for the same tile are collapsed into a single request.
func (s *Source) Tile(ctx context.Context, z, x, y uint64) ([]byte, error) {
	if s.useSingleFlight(z) {
		// Deduplicate concurrent loads of the same tile
		key := fmt.Sprintf(singleFlightKeyTemplate, s.header.Etag, z, x, y)
		data, err, _ := s.sg.Do(key, func() ([]byte, error) {
			return s.repository.Tile(ctx, s.Header(), s.reader, s.config.decompress, z, x, y)
		})
		if err != nil {
			return nil, fmt.Errorf("reading tile with singleflight: %w", err)
		}
		return data, nil
	}
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
