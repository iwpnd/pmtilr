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

// SourceOption is a functional option for configuring a Source.
type SourceOption = func(source *Source)

// WithDecompressFunc sets a custom decompression function on the Source.
func WithDecompressFunc(decompressFn DecompressFunc) SourceOption {
	return func(source *Source) {
		source.decompress = decompressFn
	}
}

// WithRepository sets a custom Repository on the Source.
func WithRepository(repository *Repository) SourceOption {
	return func(source *Source) {
		source.repository = repository
	}
}

// WithRangeReader sets a custom RangeReader on the Source.
func WithRangeReader(reader RangeReader) SourceOption {
	return func(source *Source) {
		source.reader = reader
	}
}

// Source provides read access to protomap tiles, supporting concurrent
// loads with singleflight deduplication.
type Source struct {
	reader     RangeReader    // Underlying reader for HTTP range requests
	header     *HeaderV3      // Parsed header containing tile layout and ETag
	meta       *Metadata      // Metadata for tile index and offsets
	repository *Repository    // Repository for actual tile reads
	decompress DecompressFunc // Function handling decompression on the archive
}

// NewSource initializes a Source, optionally applying SourceConfigOptions,
// and immediately loads the header and metadata.
// It returns an error if initial header or metadata reading fails.
func NewSource(ctx context.Context, uri string, options ...SourceOption) (*Source, error) {
	// Create Source with defaults
	s := &Source{
		header: &HeaderV3{},
		meta:   &Metadata{},
	}

	// apply user options
	for _, optFn := range options {
		optFn(s)
	}

	// Initialize default reader unless configured.
	if s.reader == nil {
		reader, err := NewRangeReader(ctx, uri)
		if err != nil {
			return nil, err
		}
		s.reader = reader
	}

	// Initialize default repository unless configured.
	if s.repository == nil {
		repository, err := newDefaultRepository()
		if err != nil {
			return nil, err
		}
		s.repository = repository
	}

	// Initialize default decompress function unless configured.
	if s.decompress == nil {
		s.decompress = Decompress
	}

	if err := s.header.ReadFrom(ctx, s.reader); err != nil {
		return nil, err
	}

	if err := s.meta.ReadFrom(ctx, *s.header, s.reader, s.decompress); err != nil {
		return nil, err
	}

	return s, nil
}

// Tile returns the raw tile bytes for the specified z, x, y.
func (s *Source) Tile(ctx context.Context, z, x, y uint64) ([]byte, error) {
	// NOTE: maybe validate zxy against header.bounds
	if z < uint64(s.header.MinZoom) || z > uint64(s.header.MaxZoom) {
		return []byte{}, fmt.Errorf(
			"invalid zoom: %d for allowed range of %d to %d",
			z,
			s.header.MinZoom,
			s.header.MaxZoom,
		)
	}
	return s.repository.Tile(ctx, s.Header(), s.reader, s.decompress, z, x, y)
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
