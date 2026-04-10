# pmtilr

> [!WARNING]  
> Work in progress. Might change at any time.

A standalone Go reader for [PMTiles](https://github.com/protomaps/PMTiles) that treats a tile archive like any other service (e.g., a database or HTTP client). Plug it into your handler and request tiles via `z/x/y`.

## Features

* Fast Hilbert ID resolution for quick tile look‑ups
* Default [otter/v2](https://maypok86.github.io/otter) in‑memory cache (or bring‑your‑own cache support)
* Protocol‑agnostic range reader (`file://`, `s3://`, `http(s)://` extensible)


## TODO

* add [blob](https://gocloud.dev/howto/blob/#using)

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

Comparison against [go-pmtiles](https://github.com/protomaps/go-pmtiles) serving the same PMTiles archive from local MinIO. Both servers are pinned to 2 cores (`taskset -c 0,1`), loaded with 50 concurrent users via [k6](https://k6.io/) for 15 minutes from a fixed set of tile URLs across mixed zoom levels with a consistent ~19% 404 rate.

| Metric | go-pmtiles | pmtilr | Delta |
| --- | --- | --- | --- |
| Requests/s | 6,249 | 8,622 | **+38.0%** |
| Avg latency | 7.93 ms | 5.72 ms | **-27.9%** |
| Median latency | 1.46 ms | 1.09 ms | **-25.3%** |
| P90 latency | 8.00 ms | 4.35 ms | **-45.6%** |
| P95 latency | 10.35 ms | 6.27 ms | **-39.4%** |
| Data throughput | 130 MB/s | 180 MB/s | **+38.5%** |

Tested on an Intel i7-14700KF running Linux (amd64).

## License

MIT

