package pmtilr

import (
	"strconv"

	"github.com/maypok86/otter/v2"
)

type Cacher interface {
	Get(key string) (Directory, bool)
	Set(key string, value Directory) bool
	Close()
	Clear()
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

const (
	DefaultOtterMaximumSize     = 10_000
	DefaultOtterInitialCapacity = 1_000
)

func NewOtterCache() (Cacher, error) {
	cache, err := otter.New(&otter.Options[string, Directory]{
		MaximumSize:     DefaultOtterMaximumSize,
		InitialCapacity: DefaultOtterInitialCapacity,
	})
	if err != nil {
		return nil, err
	}
	return &OtterCache{cache: cache}, nil
}

type OtterCache struct {
	cache *otter.Cache[string, Directory]
}

func (oc *OtterCache) Get(key string) (Directory, bool) {
	return oc.cache.GetIfPresent(key)
}

func (oc *OtterCache) Set(key string, value Directory) bool {
	_, ok := oc.cache.Set(key, value)

	return ok
}

func (oc *OtterCache) Close() {}

func (oc *OtterCache) Clear() {}
