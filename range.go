package pmtilr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	indexOffset                = 0
	indexSize                  = 1
	indexMinZoom               = 0
	indexMaxZoom               = 1
	singleFlightKeyTemplate    = "%d:%d:%d"
	singleFlightDefaultMinZoom = 1
	singleFlightDefaultMaxZoom = 10
)

type ZoomRange [2]uint64

func (zr ZoomRange) Validate() error {
	minZoom, maxZoom := zr[0], zr[1]
	if minZoom > maxZoom {
		return fmt.Errorf("max zoom %d cannot be bigger than min zoom %d", maxZoom, minZoom)
	}
	return nil
}

func (zr ZoomRange) MinZoom() uint64 {
	return zr[indexMinZoom]
}

func (zr ZoomRange) MaxZoom() uint64 {
	return zr[indexMaxZoom]
}

func NewZoomRange(minZoom, maxZoom uint64) ZoomRange {
	var zr ZoomRange
	zr[indexMinZoom] = minZoom
	zr[indexMaxZoom] = maxZoom
	return zr
}

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

func NewRangeReader(uri string) (RangeReader, error) {
	uri = strings.TrimSpace(uri)

	u, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, fmt.Errorf("parsing URI %q: %w", uri, err)
	}

	// FIX: resolve path from file uri
	switch u.Scheme {
	case "", "file":
		path := u.Path
		if u.Scheme == "" {
			path = uri
		}
		return NewFileRangeReader(filepath.Join(".", path))
		// TODO: extend with blob reader impls
	default:
		return nil, fmt.Errorf("unsupported URI scheme %q", u.Scheme)
	}
}

// TODO: accept uri
func NewFileRangeReader(path string) (*FileRangeReader, error) {
	filePath := filepath.Clean(path)
	f, err := os.Open(filePath)
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

	_, err := f.file.ReadAt(buf, int64(offset)) //nolint:gosec
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}

	return buf, nil
}
