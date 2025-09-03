package pmtilr

import (
	"fmt"
	"testing"
)

func TestBuildSingleflightKey(t *testing.T) {
	tests := []struct {
		name        string
		etag        string
		z, x, y     uint64
		expectedKey string
	}{
		{
			name:        "basic case",
			etag:        "abc123",
			z:           10,
			x:           512,
			y:           1024,
			expectedKey: "abc123:10:512:1024",
		},
		{
			name:        "zero values",
			etag:        "test",
			z:           0,
			x:           0,
			y:           0,
			expectedKey: "test:0:0:0",
		},
		{
			name:        "large values",
			etag:        "etag-large",
			z:           18,
			x:           262144,
			y:           131072,
			expectedKey: "etag-large:18:262144:131072",
		},
		{
			name:        "empty etag",
			etag:        "",
			z:           5,
			x:           10,
			y:           20,
			expectedKey: ":5:10:20",
		},
		{
			name:        "etag with special chars",
			etag:        "etag-with-dashes_and_underscores.123",
			z:           15,
			x:           1000,
			y:           2000,
			expectedKey: "etag-with-dashes_and_underscores.123:15:1000:2000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimizedKey := buildSingleflightKey(tt.etag, tt.z, tt.x, tt.y)
			originalKey := fmt.Sprintf(singleFlightKeyTemplate, tt.etag, tt.z, tt.x, tt.y)

			if optimizedKey != tt.expectedKey {
				t.Errorf("buildSingleflightKey() = %q, expected %q", optimizedKey, tt.expectedKey)
			}

			if optimizedKey != originalKey {
				t.Errorf("buildSingleflightKey() = %q, fmt.Sprintf() = %q, should be identical", optimizedKey, originalKey)
			}
		})
	}
}

func BenchmarkSingleflightKeyComparison(b *testing.B) {
	etag := "test-etag-12345"
	z, x, y := uint64(10), uint64(512), uint64(1024)

	b.Run("Original_fmt.Sprintf", func(b *testing.B) {
		for range b.N {
			_ = fmt.Sprintf(singleFlightKeyTemplate, etag, z, x, y)
		}
	})

	b.Run("Optimized_ByteSlicePool", func(b *testing.B) {
		for range b.N {
			_ = buildSingleflightKey(etag, z, x, y)
		}
	})
}

