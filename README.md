# pmtilr

> [!WARNING]  
> Work in progress. Might change at any time.

A standalone Go reader for [PMTiles](https://github.com/protomaps/PMTiles) that treats a tile archive like any other service (e.g., a database or HTTP client). Plug it into your handler and request tiles via `z/x/y`.

## Features

* Fast Hilbert ID resolution for quick tile look‑ups
* Default Ristretto in‑memory cache (or bring‑your‑own cache support)
* Protocol‑agnostic range reader (`file://`, `s3://`, extensible)

## Installation

```bash
go get github.com/iwpnd/pmtilr
```

## Usage

```go
package main

import (
    "context"
    "log"

    "github.com/iwpnd/pmtilr"
)

func main() {
    ctx := context.Background()

    src, err := pmtilr.NewSource(ctx, "s3://my_bucket/tiles.pmtiles")
    if err != nil {
        log.Fatalf("init source: %v", err)
    }

    // Get header as JSON
    fmt.Println(src.Header())

    // Get header as JSON string
    fmt.Println(src.Meta())

    tile, err := src.Tile(ctx, 14, 8943, 5372)
    if err != nil {
        log.Fatalf("fetch tile: %v", err)
    }

    log.Printf("tile size: %d bytes", len(tile))
}
```

## Benchmark

tbc

## License

MIT

