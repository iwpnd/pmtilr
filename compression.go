package pmtilr

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
)

type Compression uint8

const (
	CompressionUnknown = iota
	CompressionNone
	CompressionGZIP
	CompressionBrotli
	CompressionZstd
)

var compressionOptions = map[Compression]string{
	CompressionUnknown: "unknown",
	CompressionNone:    "none",
	CompressionGZIP:    "gzip",
	CompressionBrotli:  "brotli",
	CompressionZstd:    "zstd",
}

func (c Compression) String() string {
	return compressionOptions[c]
}

func (c Compression) MarshalJSON() ([]byte, error) {
	str, ok := compressionOptions[c]
	if !ok {
		str = compressionOptions[CompressionUnknown]
	}
	return json.Marshal(str)
}

type DecompressFunc = func(r io.Reader, compression Compression) (io.Reader, error)

func Decompress(r io.Reader, compression Compression) (io.Reader, error) {
	switch compression {
	case CompressionNone, CompressionUnknown:
		// No-op
		return r, nil

	case CompressionGZIP:
		if _, ok := r.(io.ByteReader); !ok {
			r = bufio.NewReader(r)
		}
		gr, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("gzip.NewReader: %w", err)
		}
		// gzip.Reader is also an io.ReadCloser, so callers can Close() when done.
		return gr, nil

	// TODO: extend
	// case CompressionBrotli:
	//     … etc …

	default:
		return nil, fmt.Errorf("unsupported compression: %v", compression)
	}
}
