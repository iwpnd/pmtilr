package pmtilr

import (
	"context"
	"fmt"
	"sync"

	singleflight "github.com/iwpnd/singleflightx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationName = "github.com/iwpnd/pmtilr"
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
	withOtel   bool

	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
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

// WithTracerProvider to pass a custom tracer provider.
func WithTracerProvider(provider trace.TracerProvider) SourceOption {
	return func(config *sourceConfig) {
		config.tracerProvider = provider
	}
}

// WithMeterProvider to pass a custom meter provider.
func WithMeterProvider(provider metric.MeterProvider) SourceOption {
	return func(config *sourceConfig) {
		config.meterProvider = provider
	}
}

// WithDisableInstrumentation disables all tracing and metrics on the pmtilr.Source.
func WithDisableInstrumentation() SourceOption {
	return func(config *sourceConfig) {
		config.withOtel = false
	}
}

type Source interface {
	Tile(ctx context.Context, z, x, y uint64) ([]byte, error)
	Header() HeaderV3
	Meta() Metadata
	TileJSON(host string) TileJSON
}

// TileSource provides read access to protomap tiles, supporting concurrent
// loads with singleflight deduplication.
type TileSource struct {
	reader     RangeReader    // Underlying reader for HTTP range requests
	header     *HeaderV3      // Parsed header containing tile layout and ETag
	meta       *Metadata      // Metadata for tile index and offsets
	repository Repository     // Repository for actual tile reads
	decompress DecompressFunc // Function handling decompression on the archive

	tracer trace.Tracer
	meter  metric.Meter
}

// NewSource initializes a Source, optionally applying SourceConfigOptions,
// and immediately loads the header and metadata.
// It returns an error if initial header or metadata reading fails.
func NewSource( //nolint:cyclop
	ctx context.Context,
	uri string,
	options ...SourceOption,
) (Source, error) {
	// Create Source with defaults
	s := &TileSource{
		header: &HeaderV3{},
		meta:   &Metadata{},
	}

	cfg := &sourceConfig{
		tracerProvider: otel.GetTracerProvider(),
		meterProvider:  otel.GetMeterProvider(),
		withOtel:       true,
	}

	// apply user options
	for _, optFn := range options {
		optFn(cfg)
	}

	tracer := cfg.tracerProvider.Tracer(instrumentationName)
	meter := cfg.meterProvider.Meter(instrumentationName)

	s.tracer = tracer
	s.meter = meter

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

	cache := cfg.cacher
	if cfg.withOtel {
		c, err := newInstrumentedCacher(cache, tracer, meter)
		if err != nil {
			return nil, fmt.Errorf("creating source: %w", err)
		}
		cache = c
	}

	repository, err := NewDirectoryRepository(cache, sg)
	if err != nil {
		return nil, err
	}
	s.repository = repository

	if cfg.withOtel {
		r, err := newInstrumentedRepository(repository, tracer, meter)
		if err != nil {
			return nil, err
		}
		s.repository = r
	}

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

	if cfg.withOtel {
		return newInstrumentedSource(s, tracer, meter)
	}

	return s, nil
}

// Tile returns the raw tile bytes for the specified z, x, y.
func (s *TileSource) Tile(ctx context.Context, z, x, y uint64) ([]byte, error) {
	// NOTE: maybe validate zxy against header.bounds
	if z < uint64(s.header.MinZoom) || z > uint64(s.header.MaxZoom) {
		return []byte{}, fmt.Errorf(
			"invalid zoom: %d for allowed range of %d to %d",
			z,
			s.header.MinZoom,
			s.header.MaxZoom,
		)
	}

	entry, err := TileEntry(ctx, s.repository, s.Header(), s.reader, s.decompress, z, x, y)
	if err != nil {
		return nil, err
	}

	return entry.ReadTileBytes(
		ctx,
		s.reader,
		s.header.TileDataOffset,
	)
}

// Header returns a copy of the current header.
func (s *TileSource) Header() HeaderV3 {
	return *s.header
}

// Meta returns a copy of the current metadata.
func (s *TileSource) Meta() Metadata {
	return *s.meta
}

// Close the source and its dependencies.
func (s *TileSource) Close() {
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
func (s *TileSource) TileJSON(host string) TileJSON {
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
