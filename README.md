# pmtilr

## Overview
pmtilr is a high-performance, standalone Golang reader for [PMTiles](https://github.com/protomaps/PMTiles). It is designed to treat a tile archive like any other service (e.g., a database or HTTP client), making it easy to integrate into existing handlers.


## Key Features
* **High Performance**: Includes fast Hilbert ID resolution for quick tile look-ups.
* **In-Memory Caching**: Uses [otter/v2](https://maypok86.github.io/otter) by default, with support for custom cache implementations.
* **Protocol Agnostic**: Supports various range readers, including `file://`, `s3://`, and `http(s)://`.
* **Observability**: Built-in support for OpenTelemetry metrics and traces.

## Installation
```bash
go get github.com/iwpnd/pmtilr
```

## Usage Example
```go
package main

import (
    "context"
    "log"
    "fmt"

    "github.com/iwpnd/pmtilr"
)

func main() {
    ctx := context.Background()

    // Initialize source (e.g., from S3)
    src, err := pmtilr.NewSource(ctx, "s3://my_bucket/tiles.pmtiles")
    if err != nil {
        log.Fatalf("init source: %v", err)
    }

    // Access metadata and headers
    fmt.Println(src.Header())
    fmt.Println(src.Meta())

    // Fetch a specific tile
    tile, err := src.Tile(ctx, 14, 8943, 5372)
    if err != nil {
        log.Fatalf("fetch tile: %v", err)
    }

    log.Printf("tile size: %d bytes", len(tile))
}
```

## Observability (OpenTelemetry)
`pmtilr` supports OpenTelemetry for both metrics and traces. By default, it uses the global OpenTelemetry provider. You can customize this behavior using the following options:

- use `WithTracerProvider(provider trace.TroperProvider)` to pass a custom tracer provider for tracing.
- use `WithMeterProvider(provider metric.MeterProvider)` to pass a custom meter provider for metrics.
- use `WithDisableInstrumentation()` to completely disable all tracing and metrics on the `pmtilr.Source`.


### Metrics
The following metrics are tracked:

- `pmtilr.source.tile.request.duration`: Histogram of tile request durations (includes `success` attribute).
- `pmtilr.directory.cache.request.duration`: Histogram of cache request durations (includes `operation` attribute).
- `pmtilr.directory.cache.hits`: Counter of cache hits (includes `cached` attribute).
- `pmtilr.repository.directory.request.duration`: Histogram of directory lookup request durations (includes `success` attribute).
- `pmtilr.repository.directory.request.shared`: Counter of requests shared via singleflight (includes `shared` and `success` attributes).
