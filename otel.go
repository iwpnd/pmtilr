package pmtilr

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func newFloat64Histogram(
	meter metric.Meter,
	name string,
	opts ...metric.Float64HistogramOption,
) (metric.Float64Histogram, error) {
	return meter.Float64Histogram(name, opts...)
}

func newInt64Counter(
	meter metric.Meter,
	name string,
	opts ...metric.Int64CounterOption,
) (metric.Int64Counter, error) {
	return meter.Int64Counter(name, opts...)
}

// instrumentedSource implements the Source interface
// and wraps a Source to collect metrics and provide tracing.
func newInstrumentedSource(
	source *TileSource,
	tracer trace.Tracer,
	meter metric.Meter,
) (Source, error) {
	requestHistogramName := "pmtilr.source.tile.request.duration"
	requestHistogram, err := newFloat64Histogram(
		meter,
		requestHistogramName,
		metric.WithDescription("tile request duration"),
	)
	if err != nil {
		return nil, fmt.Errorf("instantiating '%s' histogram: %w", requestHistogramName, err)
	}

	return &instrumentedSource{
		source:           source,
		tracer:           tracer,
		meter:            meter,
		requestHistogram: requestHistogram,
	}, nil
}

type instrumentedSource struct {
	source *TileSource

	requestHistogram metric.Float64Histogram

	tracer trace.Tracer
	meter  metric.Meter
}

func (is *instrumentedSource) Tile(ctx context.Context, z, x, y uint64) (data []byte, err error) {
	start := time.Now()
	defer func() {
		if is.requestHistogram.Enabled(ctx) {
			duration := time.Since(start)
			is.requestHistogram.Record(
				ctx,
				duration.Seconds(),
				metric.WithAttributes(
					attribute.KeyValue{Key: "success", Value: attribute.BoolValue(err == nil)},
				),
			)
		}
	}()

	return is.source.Tile(ctx, z, x, y)
}

func (is *instrumentedSource) Header() HeaderV3 {
	return is.source.Header()
}

func (is *instrumentedSource) Meta() Metadata {
	return is.source.Meta()
}

func (is *instrumentedSource) Close() {
	is.source.Close()
}

func (is *instrumentedSource) TileJSON(host string) TileJSON {
	return is.source.TileJSON(host)
}

// instrumentedCacher satisfied the Cacher interface,
// and wraps a Cacher to collect metrics and provide tracing.
type instrumentedCacher struct {
	cache Cacher

	requestHistogram metric.Float64Histogram
	cacheHitCounter  metric.Int64Counter

	tracer trace.Tracer
	meter  metric.Meter
}

func newInstrumentedCacher(
	cache Cacher,
	tracer trace.Tracer,
	meter metric.Meter,
) (*instrumentedCacher, error) {
	requestHistogramName := "pmtilr.directory.cache.request.duration"
	requestHistogram, err := newFloat64Histogram(
		meter,
		requestHistogramName,
		metric.WithDescription("cache request duration"),
	)
	if err != nil {
		return nil, fmt.Errorf("instantiating '%s' histogram: %w", requestHistogramName, err)
	}

	cacheHitCounterName := "pmtilr.directory.cache.hits"
	cacheHitCounter, err := newInt64Counter(
		meter,
		cacheHitCounterName,
		metric.WithDescription("pmtilr repository in-memory directory cache hits"),
	)
	if err != nil {
		return nil, fmt.Errorf("instantiating '%s' counter: %w", cacheHitCounterName, err)
	}
	return &instrumentedCacher{
		cache:  cache,
		tracer: tracer,
		meter:  meter,

		requestHistogram: requestHistogram,
		cacheHitCounter:  cacheHitCounter,
	}, nil
}

func (ic *instrumentedCacher) Get(ctx context.Context, key string) (Directory, bool) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if ic.requestHistogram.Enabled(ctx) {
			ic.requestHistogram.Record(
				ctx,
				duration.Seconds(),
				metric.WithAttributes(
					attribute.KeyValue{Key: "operation", Value: attribute.StringValue("get")},
				),
			)
		}
	}()

	dir, cached := ic.cache.Get(ctx, key)

	if ic.cacheHitCounter.Enabled(ctx) {
		ic.cacheHitCounter.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.KeyValue{Key: "cached", Value: attribute.BoolValue(cached)},
			),
		)
	}

	return dir, cached
}

func (ic *instrumentedCacher) Set(ctx context.Context, key string, value Directory) bool {
	start := time.Now()
	defer func() {
		if ic.requestHistogram.Enabled(ctx) {
			duration := time.Since(start)
			ic.requestHistogram.Record(
				ctx,
				duration.Seconds(),
				metric.WithAttributes(
					attribute.KeyValue{Key: "operation", Value: attribute.StringValue("set")},
				),
			)
		}
	}()

	return ic.cache.Set(ctx, key, value)
}

func (ic *instrumentedCacher) Close() {
	ic.cache.Close()
}

func (ic *instrumentedCacher) Clear() {
	ic.cache.Clear()
}

// instrumentedRepository satisfies the Repository interface
// and wraps a Repository to collect metrics and provide tracing.
type instrumentedRepository struct {
	repository Repository

	sharedRequestCounter   metric.Int64Counter
	sharedRequestHistogram metric.Float64Histogram

	tracer trace.Tracer
	meter  metric.Meter
}

func newInstrumentedRepository(
	repository Repository,
	tracer trace.Tracer,
	meter metric.Meter,
) (*instrumentedRepository, error) {
	sharedRequestDurationHistogramName := "pmtilr.repository.directory.request.duration"
	sharedRequestHistogram, err := newFloat64Histogram(
		meter,
		sharedRequestDurationHistogramName,
		metric.WithDescription("directory lookup request duration"),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"instantiating '%s' metric: %w",
			sharedRequestDurationHistogramName,
			err,
		)
	}

	sharedRequestCounterName := "pmtilr.repository.directory.request.shared"
	sharedRequestCounter, err := newInt64Counter(
		meter,
		sharedRequestCounterName,
		metric.WithDescription("request shared in singleflight request"),
	)
	if err != nil {
		return nil, fmt.Errorf("instantiating '%s' metric: %w", sharedRequestCounterName, err)
	}

	return &instrumentedRepository{
		repository:             repository,
		tracer:                 tracer,
		meter:                  meter,
		sharedRequestCounter:   sharedRequestCounter,
		sharedRequestHistogram: sharedRequestHistogram,
	}, nil
}

func (ir *instrumentedRepository) Close() {
	ir.repository.Close()
}

func (ir *instrumentedRepository) DirectoryAt(
	ctx context.Context,
	header HeaderV3,
	reader RangeReader,
	ranger Ranger,
	decompress DecompressFunc,
) (dir Directory, shared bool, err error) {
	start := time.Now()
	defer func() {
		if ir.sharedRequestHistogram.Enabled(ctx) {
			duration := time.Since(start)
			ir.sharedRequestHistogram.Record(
				ctx,
				duration.Seconds(),
				metric.WithAttributes(
					attribute.KeyValue{Key: "success", Value: attribute.BoolValue(err == nil)},
				),
			)
		}
	}()

	dir, shared, err = ir.repository.DirectoryAt(ctx, header, reader, ranger, decompress)
	if ir.sharedRequestCounter.Enabled(ctx) {
		ir.sharedRequestCounter.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.KeyValue{
					Key:   "shared",
					Value: attribute.BoolValue(shared),
				},
				attribute.KeyValue{Key: "success", Value: attribute.BoolValue(err == nil)},
			),
		)
	}

	return dir, shared, err
}
