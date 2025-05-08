package pmtilr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

type Sizer interface {
	Size() uint64
}

type Offsetter interface {
	Offset() uint64
}

type Ranger interface {
	Offsetter
	Sizer
	Validate() error
}

const (
	indexOffset = 0
	indexSize   = 1
)

type Range [2]uint64

func (r Range) Offset() uint64 {
	return r[indexOffset]
}

func (r Range) Size() uint64 {
	return r[indexSize]
}

func (r Range) Validate() error {
	if r.Size() == 0 {
		return errors.New("invalid range. size must be a positiv integer")
	}
	return nil
}

func NewRange(offset, size uint64) Range {
	var r Range
	r[indexOffset] = offset
	r[indexSize] = size
	return r
}

type RangeReader interface {
	ReadRange(ctx context.Context, ranger Ranger) ([]byte, error)
	// TODO:
	// read multiple ranges ReadRanges(ctx, ranges []Ranger)
}

// TODO: accept uri
func NewFileRangeReader(path string) (*FileRangeReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file at path %s: %w", path, err)
	}

	return &FileRangeReader{file: f}, nil
}

type FileRangeReader struct {
	file io.ReaderAt
}

func (f *FileRangeReader) ReadRange(ctx context.Context, ranger Ranger) ([]byte, error) {
	if err := ranger.Validate(); err != nil {
		return []byte{}, fmt.Errorf("invalid ranger: %w", err)
	}

	offset := ranger.Offset()
	size := ranger.Size()
	buf := make([]byte, size)

	_, err := f.file.ReadAt(buf, int64(offset))
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	return buf, nil
}

type SourceConfig struct {
	decompress DecompressFunc
}

type SourceConfigOption = func(config *SourceConfig)

func WithCustomDecompressFunc(decompressFn DecompressFunc) SourceConfigOption {
	return func(config *SourceConfig) {
		config.decompress = decompressFn
	}
}

type Source struct {
	reader     RangeReader
	header     HeaderV3
	meta       Metadata
	config     *SourceConfig
	repository *Repository
}

func NewSource(reader RangeReader, options ...SourceConfigOption) (*Source, error) {
	s := &Source{
		reader: reader,
		header: HeaderV3{},
		meta:   Metadata{},
	}

	config := &SourceConfig{
		decompress: Decompress,
	}

	for _, o := range options {
		o(config)
	}

	if err := s.header.ReadFrom(s.reader); err != nil {
		return nil, err
	}

	if err := s.meta.ReadFrom(s.header, s.reader, s.config.decompress); err != nil {
		return nil, err
	}

	repo, err := NewRepository()
	if err != nil {
		return nil, err
	}

	s.repository = repo

	return s, nil
}

func (s *Source) Tile(ctx context.Context, z, x, y uint64) ([]byte, error) {
	return s.repository.Tile(ctx, s.header, s.reader, s.config.decompress, z, x, y)
}

func (s *Source) Header() HeaderV3 {
	return s.header
}

func (s *Source) Meta() Metadata {
	return s.meta
}
