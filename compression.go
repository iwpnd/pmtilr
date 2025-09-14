package pmtilr

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// Compression enumerates supported compression codecs for PMTiles content.
// The zero value is CompressionUnknown.
type Compression uint8

const (
	// CompressionUnknown indicates the codec is not recognized.
	CompressionUnknown = iota
	// CompressionNone indicates the payload is uncompressed.
	CompressionNone
	// CompressionGZIP indicates the payload is gzip-compressed.
	CompressionGZIP
	// CompressionBrotli indicates the payload is Brotli-compressed.
	CompressionBrotli
	// CompressionZstd indicates the payload is Zstandard-compressed.
	CompressionZstd
)

// compressionOptions maps Compression to a human-readable name.
// String() and MarshalJSON use this table.
var compressionOptions = map[Compression]string{
	CompressionUnknown: "unknown",
	CompressionNone:    "none",
	CompressionGZIP:    "gzip",
	CompressionBrotli:  "brotli",
	CompressionZstd:    "zstd",
}

// String returns a human-readable name for the compression algorithm.
// Unknown values are rendered as "unknown".
func (c Compression) String() string {
	return compressionOptions[c]
}

// MarshalJSON marshals the Compression as a JSON string (e.g. "gzip").
// Unknown values marshal as "unknown".
func (c Compression) MarshalJSON() ([]byte, error) {
	str, ok := compressionOptions[c]
	if !ok {
		str = compressionOptions[CompressionUnknown]
	}
	return json.Marshal(str)
}

// DecompressFunc is a function that wraps an io.ReadCloser with the
// appropriate decompressor for the given Compression. The returned
// io.ReadCloser must be closed by the caller to release resources.
type DecompressFunc = func(r io.ReadCloser, compression Compression) (io.ReadCloser, error)

// gzPool stores reusable *gzip.Reader instances to reduce allocations.
// gzip.Reader is not safe for concurrent use, but sync.Pool access is
// concurrency-safe and returns a fresh instance per caller.
var gzPool = sync.Pool{New: func() any { return new(gzip.Reader) }}

// GZIPReadCloser wraps a gzip reader together with a Closer. Closing the
// GZIPReadCloser closes the gzip reader first and then the underlying
// source (e.g., an S3 body).
type GZIPReadCloser struct {
	io.Reader
	io.Closer
}

// closeFunc adapts a func() error to io.Closer.
type closeFunc func() error

// Close implements io.Closer.
func (f closeFunc) Close() error { return f() }

// NewGZIPReadCloser returns a pooled gzip reader that reads from rc.
// The returned ReadCloser must be closed; on Close it will:
//  1. Close the gzip reader,
//  2. Return it to the pool, and
//  3. Close the underlying rc.
//
// Errors:
//   - If the gzip reader cannot be initialized (Reset fails), rc is closed
//     and the error is returned.
func NewGZIPReadCloser(rc io.ReadCloser) (io.ReadCloser, error) {
	zr, _ := gzPool.Get().(*gzip.Reader) //nolint:errcheck
	if err := zr.Reset(rc); err != nil {
		gzPool.Put(zr)
		_ = rc.Close() //nolint:errcheck // ensure underlying is closed on init failure
		return nil, err
	}
	return GZIPReadCloser{
		Reader: zr,
		Closer: closeFunc(func() error {
			cerr := zr.Close()
			gzPool.Put(zr)
			return errors.Join(cerr, rc.Close())
		}),
	}, nil
}

// Decompress wraps r with a decompressor based on the provided Compression.
//
// Behavior:
//   - CompressionNone, CompressionUnknown: r is returned unchanged. The caller
//     is still responsible for calling Close on the returned ReadCloser.
//   - CompressionGZIP: returns a pooled gzip ReadCloser that owns r and must
//     be closed by the caller (which will, in turn, close r).
//   - Other codecs: currently unsupported; returns an error.
func Decompress(r io.ReadCloser, compression Compression) (io.ReadCloser, error) {
	switch compression {
	case CompressionNone, CompressionUnknown:
		return r, nil

	case CompressionGZIP:
		gr, err := NewGZIPReadCloser(r)
		if err != nil {
			return nil, fmt.Errorf("gzip.NewReader: %w", err)
		}
		return gr, nil

	// TODO: extend
	// case CompressionBrotli:
	//   return NewBrotliReadCloser(r)
	// case CompressionZstd:
	//   return NewZstdReadCloser(r)

	default:
		return nil, fmt.Errorf("unsupported compression: %v", compression)
	}
}
