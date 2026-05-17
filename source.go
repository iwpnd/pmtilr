package pmtilr

import (
	"context"
	"fmt"
	"sync"

	singleflight "github.com/iwpnd/singleflightx"
)

// keyBufPool provides a shared buffer pool with 64-byte pre-allocated buffers
var keyBufPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 64) // Pre-allocate 64 bytes capacity, sufficient for key pattern
		return &buf
	},
}

type sourceConfig struct {
	reader     RangeReader
	cacher     Cacher
	decompress DecompressFunc
	sfxshards  uint64
}

// SourceOption is a functional option for configuring a Source.
type SourceOption = func(source *sourceConfig)

// WithDecompressFunc sets a custom decompression function on the Source.
func WithDecompressFunc(decompressFn DecompressFunc) SourceOption {
	return func(config *sourceConfig) {
		config.decompress = decompressFn
	}
}

// WithCacher sets a custom in directory cache on the Source.
func WithCacher(cacher Cacher) SourceOption {
	return func(config *sourceConfig) {
		config.cacher = cacher
	}
}

// WithRangeReader sets a custom RangeReader on the Source.
func WithRangeReader(reader RangeReader) SourceOption {
	return func(config *sourceConfig) {
		config.reader = reader
	}
}

// WithSingleFlightShardCount change the number of singleflight shards from default 3.
func WithSingleFlightShardCount(shards uint64) SourceOption {
	return func(config *sourceConfig) {
		config.sfxshards = shards
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

	cfg := &sourceConfig{}

	// apply user options
	for _, optFn := range options {
		optFn(cfg)
	}

	if cfg.cacher == nil {
		cache, err := NewOtterCache()
		if err != nil {
			return nil, err
		}
		cfg.cacher = cache
	}

	if cfg.sfxshards == 0 {
		cfg.sfxshards = defaultSfxShardCount
	}

	s.reader = cfg.reader
	// Initialize default reader unless configured.
	if s.reader == nil {
		reader, err := NewRangeReader(ctx, uri)
		if err != nil {
			return nil, err
		}
		s.reader = reader
	}

	sg := singleflight.NewShardedGroup[string, Directory](
		singleflight.WithShardCount(cfg.sfxshards),
	)
	repository, err := NewRepository(cfg.cacher, sg)
	if err != nil {
		return nil, err
	}
	s.repository = repository

	s.decompress = cfg.decompress
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

type TileJSON struct {
	TileJSON     string        `json:"tilejson"`
	Name         string        `json:"name,omitempty"`
	Description  string        `json:"description,omitempty"`
	Attribution  string        `json:"attribution,omitempty"`
	Scheme       string        `json:"scheme"`
	Tiles        []string      `json:"tiles"`
	VectorLayers []VectorLayer `json:"vector_layers,omitempty"`
}

// TileJSON produces a TileJSON document from the archive metadata.
// For MVT and MLT tile types TileJSON v3 (with vector_layers);
// else return TileJSON v2.
func (s *Source) TileJSON(host string) TileJSON {
	tileURL := fmt.Sprintf(
		"%s/{z}/{x}/{y}%s",
		host, s.Header().TileType.Ext(),
	)

	m := s.Meta()
	tj := TileJSON{
		Name:        m.Name,
		Description: m.Description,
		Attribution: m.Attribution,
		Scheme:      "xyz",
		Tiles:       []string{tileURL},
	}

	if s.Header().TileType.IsVector() {
		tj.TileJSON = "3.0.0"
		tj.VectorLayers = m.VectorLayers
	} else {
		tj.TileJSON = "2.2.0"
	}

	return tj
}
