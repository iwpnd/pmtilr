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

## Source API

In addition to `Tile()`, `Header()`, and `Meta()`, the `Source` interface provides:

- `TileJSON(host string) TileJSON`: generates a [TileJSON](https://github.com/mapbox/tilejson-spec) v2 or v3 document from archive metadata (v3 with `vector_layers` for MVT/MLT types).
- `Close()`: releases underlying resources (cache, connections).

If a tile is not present in the archive, `Tile()` returns `pmtilr.ErrTileNotFound`.

## Tile Types

The `TileType` enum identifies the format of tiles in the archive:

| Constant | Value | Extension | Content Type |
|----------|-------|-----------|-------------|
| `TileTypeMVT` | 1 | `.mvt` | `application/x-protobuf` |
| `TileTypePNG` | 2 | `.png` | `image/png` |
| `TileTypeJPEG` | 3 | `.jpeg` | `image/jpeg` |
| `TileTypeWebp` | 4 | `.webp` | `image/webp` |
| `TileTypeAvif` | 5 | `.avif` | `image/avif` |
| `TileTypeMLT` | 6 | `.mlt` | `application/vnd.maplibre-vector-tile` |

Helper methods:
- `Ext()` returns the file extension (e.g. `.mvt`).
- `ToContentType()` returns the HTTP Content-Type string.
- `IsVector()` returns `true` for MVT and MLT types.

## Range Readers

`pmtilr` ships with four built-in `RangeReader` implementations:

- `NewFileRangeReader(path)`: reads from local files.
- `NewMMapFileRangeReader(path)`: memory-mapped local file access for lower latency on repeated reads.
- `NewHTTPRangeReader(host, ...opts)`: HTTP/HTTPS range requests via `rip.Client`.
- `NewS3RangeReader(bucket, key, client)`: S3 range requests via the AWS SDK.

Pass a custom reader with `WithRangeReader(reader)` to override the default, or implement the `RangeReader` interface for any backend.

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

## Development

### Prerequisites
- Go 1.26+
- Docker (for S3 integration tests via MinIO)

### Commands
```bash
make test    # run tests with verbose output
make lint    # run golangci-lint
make dev-up  # start MinIO dev environment
make dev-down # stop MinIO dev environment
```

### Pre-commit Hooks
This repository uses [pre-commit](https://pre-commit.com) and enforces conventional commits with [gitlint](https://jorisroovers.github.io/gitlint). Install with:

```bash
pre-commit install
```

### Commit Convention
Commits follow [Conventional Commits](https://www.conventionalcommits.org/) and are validated by gitlint. Releases are automated via semantic-release.
