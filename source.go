package pmtilr

import (
	"context"
	"fmt"
	"sync"

	"github.com/brunomvsouza/singleflight"
)

type SourceConfig struct {
	decompress            DecompressFunc
	singleFlightZoomRange ZoomRange
}

type SourceConfigOption = func(config *SourceConfig)

func WithCustomDecompressFunc(decompressFn DecompressFunc) SourceConfigOption {
	return func(config *SourceConfig) {
		config.decompress = decompressFn
	}
}

func WithSingleFlightZoomRange(zrange [2]uint64) SourceConfigOption {
	return func(config *SourceConfig) {
		config.singleFlightZoomRange = zrange
	}
}

type Source struct {
	reader     RangeReader
	header     *HeaderV3
	meta       *Metadata
	config     *SourceConfig
	repository *Repository

	sg *singleflight.Group[string, []byte]
	mu sync.Mutex
}

func NewSource(reader RangeReader, options ...SourceConfigOption) (*Source, error) {
	config := &SourceConfig{
		decompress:            Decompress,
		singleFlightZoomRange: NewZoomRange(singleFlightDefaultMinZoom, singleFlightDefaultMaxZoom),
	}

	for _, o := range options {
		o(config)
	}

	err := config.singleFlightZoomRange.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid single flight zoom range: %w", err)
	}

	s := &Source{
		reader: reader,
		header: &HeaderV3{},
		meta:   &Metadata{},
		config: config,
		sg:     &singleflight.Group[string, []byte]{},
	}

	if err := s.updateHeader(); err != nil {
		return nil, err
	}

	if err := s.updateMeta(); err != nil {
		return nil, err
	}

	repo, err := NewRepository()
	if err != nil {
		return nil, err
	}

	s.repository = repo

	return s, nil
}

func (s *Source) useSingleFlight(z uint64) bool {
	return z >= s.config.singleFlightZoomRange.MinZoom() &&
		z <= s.config.singleFlightZoomRange.MaxZoom()
}

func (s *Source) updateHeader() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.header.ReadFrom(s.reader)
}

func (s *Source) updateMeta() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.meta.ReadFrom(*s.header, s.reader, s.config.decompress)
}

func (s *Source) Tile(ctx context.Context, z, x, y uint64) ([]byte, error) {
	// TODO: metrics for shared calls
	if s.useSingleFlight(z) {
		data, err, _ := s.sg.Do(
			fmt.Sprintf(singleFlightKeyTemplate, z, x, y),
			func() ([]byte, error) {
				return s.repository.Tile(ctx, s.Header(), s.reader, s.config.decompress, z, x, y)
			},
		)
		if err != nil {
			return nil, fmt.Errorf("reading tile with singleflight group: %w", err)
		}
		return data, err
	}
	return s.repository.Tile(ctx, s.Header(), s.reader, s.config.decompress, z, x, y)
}

func (s *Source) Header() HeaderV3 {
	return *s.header
}

func (s *Source) Meta() Metadata {
	return *s.meta
}
