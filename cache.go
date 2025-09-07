package pmtilr

import (
	"strconv"

	"github.com/dgraph-io/ristretto/v2"
)

const (
	DefaultRistrettoNumCounters = 10 * 500 * 1024
	DefaultRistrettoMaxCost     = 50 * 1024
	DefaultRistrettoBufferItems = 64
)

type Cacher interface {
	Get(key string) (Directory, bool)
	Set(key string, value Directory) bool
	Close()
	Clear()
}

func NewRistrettoCache(opts ...RistrettoCacheOption) (*RistrettoCache, error) {
	cfg := &ristretto.Config[string, Directory]{
		NumCounters: DefaultRistrettoNumCounters,
		MaxCost:     DefaultRistrettoMaxCost,
		BufferItems: DefaultRistrettoBufferItems,
	}

	for _, o := range opts {
		o(cfg)
	}

	cache, err := ristretto.NewCache(cfg)
	if err != nil {
		return &RistrettoCache{}, err
	}

	return &RistrettoCache{
		cache: cache,
	}, nil
}

// buildCacheKey efficiently builds a singleflight key using a shared buffer pool
func buildCacheKey(etag string, offset, length uint64) string {
	bufPtr, _ := keyBufPool.Get().(*[]byte) //nolint:errcheck
	buf := (*bufPtr)[:0]                    // Reset length but keep capacity
	defer keyBufPool.Put(bufPtr)

	buf = append(buf, etag...)
	buf = append(buf, ':')
	buf = strconv.AppendUint(buf, offset, 10)
	buf = append(buf, ':')
	buf = strconv.AppendUint(buf, length, 10)

	return string(buf)
}

type RistrettoCache struct {
	cache *ristretto.Cache[string, Directory]
}

type RistrettoCacheOption = func(
	rc *ristretto.Config[string, Directory],
) func(rc *ristretto.Config[string, Directory])

func (rc *RistrettoCache) Get(key string) (Directory, bool) {
	return rc.cache.Get(key)
}

func (rc *RistrettoCache) Set(key string, value Directory) bool {
	ok := rc.cache.Set(key, value, 1)
	rc.cache.Wait()

	return ok
}

func (rc *RistrettoCache) Close() {
	rc.cache.Close()
}

func (rc *RistrettoCache) Clear() {
	rc.cache.Clear()
}
