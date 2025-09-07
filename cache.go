package pmtilr

import (
	"os"
	"strconv"

	"github.com/dgraph-io/ristretto/v2"
)

const (
	DefaultRistrettoNumCounters = 10 * 500 * 1024
	DefaultRistrettoMaxCost     = 50 * 1024
	DefaultRistrettoBufferItems = 64

	cacheKeyTemplate = "%s:%d:%d" // etag:offset:size
)

func NewDefaultCache() (*ristretto.Cache[string, Directory], error) {
	cfg := &ristretto.Config[string, Directory]{
		NumCounters: getEnv(
			"PMTILR_RISTRETTO_NUM_COUNTERS",
			DefaultRistrettoNumCounters,
		), // number of keys to track frequency of (10M).
		MaxCost: getEnv(
			"PMTILR_RISTRETTO_MAX_COST",
			DefaultRistrettoMaxCost,
		), // 500mb
		BufferItems: getEnv(
			"PMTILR_RISTRETTO_BUFFER_ITEMS",
			DefaultRistrettoBufferItems,
		), // number of keys per Get buffer.
	}

	cache, err := ristretto.NewCache(cfg)
	if err != nil {
		return nil, err
	}

	return cache, nil
}

func getEnv(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fallback
		}
		return i
	}
	return fallback
}
