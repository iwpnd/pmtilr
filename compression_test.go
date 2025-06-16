package pmtilr

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestDecompress(t *testing.T) {
	tests := []struct {
		name        string
		compression Compression
		input       string
		expectError bool
	}{
		{
			name:        "No compression",
			compression: CompressionNone,
			input:       "test-data",
			expectError: false,
		},
		{
			name:        "Unknown compression",
			compression: CompressionUnknown,
			input:       "test-data",
			expectError: false,
		},
		{
			name:        "GZIP compression",
			compression: CompressionGZIP,
			input:       "test-data",
			expectError: false,
		},
		{
			name:        "Unsupported compression",
			compression: CompressionBrotli,
			input:       "test-data",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			var r io.Reader

			if tc.compression == CompressionGZIP {
				gw := gzip.NewWriter(&buf)
				_, _ = gw.Write([]byte(tc.input))
				_ = gw.Close()
				r = &buf
			} else {
				r = bytes.NewReader([]byte(tc.input))
			}

			dr, err := Decompress(r, tc.compression)
			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			out, err := io.ReadAll(dr)
			if err != nil {
				t.Fatalf("reading decompressed data: %v", err)
			}

			if string(out) != tc.input {
				t.Errorf("got %q, want %q", string(out), tc.input)
			}
		})
	}
}
