package pmtilr

import "testing"

func TestBuildCacheKey(t *testing.T) {
	type tcase struct {
		offset, length uint64
		etag, expected string
	}

	tests := map[string]tcase{
		"w/o prefix": {
			etag:   "bar",
			offset: 0, length: 0,
			expected: "bar:0:0",
		},
	}

	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			got := buildCacheKey(tt.etag, tt.offset, tt.length)

			if got != tt.expected {
				t.Errorf("expected: %s, but got: %s", tt.expected, got)
			}
		})
	}
}
