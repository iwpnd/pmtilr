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

// instrumentedCacher satisfied the Cacher interface
// that collects metrics and provides tracing.
type instrumentedCacher struct {
	cache Cacher

	requestHistogram metric.Float64Histogram
	cacheMissCounter metric.Int64Counter

	tracer trace.Tracer
	meter  metric.Meter
}

func newInstrumentedCacher(
	cache Cacher,
	tracer trace.Tracer,
	meter metric.Meter,
) (*instrumentedCacher, error) {
	requestHistogramName := "cache.duration"
	requestHistogram, err := newFloat64Histogram(
		meter,
		requestHistogramName,
		metric.WithDescription("cache request duration"),
	)
	if err != nil {
		return nil, fmt.Errorf("instantiating '%s' histogram: %w", requestHistogramName, err)
	}

	cacheMissCounterName := "cache.miss"
	cacheMissCounter, err := newInt64Counter(
		meter,
		cacheMissCounterName,
		metric.WithDescription("cache miss"),
	)
	if err != nil {
		return nil, fmt.Errorf("instantiating '%s' counter: %w", cacheMissCounterName, err)
	}
	return &instrumentedCacher{
		cache:  cache,
		tracer: tracer,
		meter:  meter,

		requestHistogram: requestHistogram,
		cacheMissCounter: cacheMissCounter,
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

	if ic.cacheMissCounter.Enabled(ctx) {
		ic.cacheMissCounter.Add(
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

// // instrumentedRepository sat
// // that collects metrics and provides tracing.
// type instrumentedRepository struct {
// 	repository *Repository
//
// 	tracer trace.Tracer
// 	meter  metric.Meter
// }
//
// func newInstrumentedRepository(
// 	repository *Repository,
// 	tracer trace.Tracer,
// 	meter metric.Meter,
// ) *instrumentedRepository {
// 	return &instrumentedRepository{
// 		repository: repository,
// 		tracer:     tracer,
// 		meter:      meter,
// 	}
// }
