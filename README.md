# pmtilr

A standalone Go reader for [PMTiles](https://github.com/protomaps/PMTiles) that treats a tile archive like any other service (e.g., a database or HTTP client). Plug it into your handler and request tiles via `z/x/y`.

## Features

* Fast Hilbert ID resolution for quick tile look‑ups
* Default Ristretto in‑memory cache (or bring‑your‑own cache support)
* Protocol‑agnostic range reader (`file://`, `s3://`, extensible)
* (optional) `singleflight` deduplication of concurrent requests

## Installation

```bash
go get github.com/yourname/pmtilr
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

    tile, err := src.Tile(ctx, 14, 8943, 5372)
    if err != nil {
        log.Fatalf("fetch tile: %v", err)
    }

    log.Printf("tile size: %d bytes", len(tile))
}
```

## Config Options

```go
// zoom ranges (min to max) that are deduplicated by singleflight
pmtilr.WithSingleFlightZoomRange([2]uint64{0, 14})
```

## Benchmark

100 VUs · local MinIO backend · same tile archive

```
PU:    Intel(R) Core(TM) i7-14700KF
Cores:  28
RAM: 62.6 GiB
OS:     EndeavourOS
Kernel: 6.15.8-arch1-1
```

| Metric                              |      **pmtilr** | **go‑pmtiles** | Relative Δ                      |
| ----------------------------------- | --------------: | -------------: | :------------------------------ |
| **Throughput (RPS)**                | **6 826 req/s** |    2 789 req/s | **+145 %** more requests/second |
| **Average latency**                 |     **14.6 ms** |        35.8 ms | **‑59 %** lower                 |
| P90 latency                         |     **43.0 ms** |       110.8 ms | **‑61 %** lower                 |
| P95 latency                         |     **60.3 ms** |       158.2 ms | **‑62 %** lower                 |
| Data received                       |          7.4 GB |         2.0 GB | +270 % (matches higher RPS)     |
| 2xx responses                       |         383 221 |        156 795 | —                               |
| 204 responses\*                     |          26 465 |         10 651 | —                               |
| **Avg CPU** (% of one core)         |           177 % |          125 % | **+42 %** CPU used              |
| **Peak RSS**                        |           83 MB |          79 MB | +4 MB                           |


## License

MIT

