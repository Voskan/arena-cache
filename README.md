# arena-cache: High-Performance, GC-Free In-Process Cache for Go 1.24 with Arena Allocator

> **GC‑free, high‑throughput in‑process cache for Go 1.24, powered by the new `arena` allocator**

[![CI](https://github.com/Voskan/arena-cache/actions/workflows/ci.yml/badge.svg)](https://github.com/Voskan/arena-cache/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Voskan/arena-cache.svg)](https://pkg.go.dev/github.com/Voskan/arena-cache)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

`arena-cache` provides an **O(1)** generational cache with near‑zero garbage‑collector overhead. It exploits Go 1.24's _stable_ `arena` API, CLOCK‑Pro replacement, sharding, and Prometheus metrics — all packed into a minimal import‑one‑package experience.

---

## ✨ Highlights

| Feature                          | Details                                                                             |
| -------------------------------- | ----------------------------------------------------------------------------------- |
| **0 GC allocations on hot path** | Values live in arenas outside the managed heap. Eviction = `arena.Free()` (O(1)).   |
| **TTL & capacity eviction**      | Generational ring (TTL) + CLOCK‑Pro (capacity) give predictable latency under load. |
| **Sharded concurrency**          | Lock contention is negligible: N shards × RWMutex.                                  |
| **Pluggable weight & callbacks** | `WithWeightFn` for custom cost; `WithEjectCallback` for L2 caches (disk, Redis…).   |
| **Metrics & tracing**            | Prometheus counters/gauges; OpenTelemetry spans in all public APIs.                 |
| **Inspector CLI**                | `arena-cache-inspect` fetches live stats, dumps pprof, works in Docker/K8s.         |
| **Tiny binaries & images**       | Static musl build ≈4 MiB; scratch Docker image ≈9 MiB.                              |

---

## 🚀 Quick Start

```bash
go get github.com/Voskan/arena-cache@latest
```

```go
package main

import (
    "context"
    "fmt"
    "time"

    cache "github.com/Voskan/arena-cache/pkg"
)

type user struct{ ID, Name string }

func main() {
    // 128 MiB capacity per instance, 10‑min TTL, 16 shards.
    c, _ := cache.New[string, user](128<<20, 10*time.Minute, 16)

    // Put
    c.Put(context.Background(), "u123", user{"u123", "Ada"}, 1)

    // GetOrLoad (singleflight‑deduplicated)
    u, _ := c.GetOrLoad(context.Background(), "u999", func(ctx context.Context, k string) (user, error) {
        // e.g. fetch from DB
        return user{ID: k, Name: "generated"}, nil
    })
    fmt.Println(u)
}
```

---

## 🛠️ Inspector CLI

```bash
# Install (via Go install or GitHub Release binaries)
go install github.com/Voskan/arena-cache/cmd/arena-cache-inspect@latest

arena-cache-inspect -addr http://localhost:6060           # one‑shot
arena-cache-inspect -watch -interval 5s                   # streaming
arena-cache-inspect -heap heap.out                        # download pprof
```

The target service must expose:

- `/debug/arena-cache/snapshot` → JSON with stats (provided by examples & README snippets)
- `/metrics` (optional) → Prometheus exposition

---

## 📊 Benchmarks

| Benchmark (Go 1.24, 24‑core AMD EPYC) | arena‑cache | Ristretto | Δ               |
| ------------------------------------- | ----------- | --------- | --------------- |
| `Get` p99 latency                     | **45 ns**   | 310 ns    | **6.8× faster** |
| Allocations/op                        | **0**       | 0.5       | —               |
| GC pause @ 1 M RPS (99p)              | **0 µs**    | 6 ms      | n/a             |

> Full reproduction: `go test ./bench -bench=. -benchmem` (see `bench/bench_test.go`).

---

## 🔍 Architecture Overview

```
┌─────────┐ requests ┌──────────┐   CLOCK-Pro ┌──────────┐ rotate  ┌──────────┐ free ┌───────┐
│  App    ├─────────▶│  Shard N ├────────────▶│  Shard 0 ├────────▶│  Arena 0 ├──────▶│  GC  │
└─────────┘          └──────────┘   …         └──────────┘         └──────────┘      └───────┘
      ▲                      ▲                                             │
      │                      └── Prom metrics, OTel traces ▲                │
      │                                           snapshot │                │
      └──────────────────────────────────────── arena-cache-inspect ◀────────┘
```

- **Shards** hold their own key→entry map and CLOCK‑Pro ring.
- **Generations** are arenas in a circular buffer; rotation by TTL or capacity.
- **Inspector** talks HTTP to emit stats or download pprof profiles.

Detailed diagrams live in [`docs/architecture.md`](docs/architecture.md).

---

## 🐳 Docker / Compose

```bash
docker compose up --build   # demo + Prometheus + inspector
```

- **demo** service runs `examples/basic` on port 6060.
- **prometheus** scrapes `/metrics`; UI on [http://localhost:9090](http://localhost:9090).
- **inspector** prints live stats every 2s.

---

## 📦 Binaries & Packages

| Platform        | Asset                                           |
| --------------- | ----------------------------------------------- |
| `linux/amd64`   | `arena-cache-inspect_<ver>_Linux_x86_64.tar.gz` |
| `linux/arm64`   | `..._Linux_arm64.tar.gz`                        |
| `darwin/arm64`  | `..._macOS_arm64.tar.gz`                        |
| `windows/amd64` | `..._Windows_x86_64.zip`                        |

Download from the [GitHub Releases](https://github.com/Voskan/arena-cache/releases) page or via `go install`.

---

## 🧑‍💻 Development

```bash
make all        # lint + test
make bench      # performance
make docs-serve # live MkDocs preview
```

- CI: GitHub Actions (`.github/workflows/ci.yml`).
- Release: tag `vX.Y.Z` → binaries, Docker images, Homebrew formula.
- Security: weekly CodeQL + Dependabot.

### Contributing

PRs & issues are welcome! Please run `make lint test` before pushing.

---

## 🗺️ Roadmap

- [x] MVP (arena, CLOCK‑Pro, CLI, metrics)
- [ ] Adaptive rotation based on access pattern
- [ ] Tiered arenas per priority class
- [ ] Native exporter for OpenTelemetry metrics
- [ ] gRPC API for remote snapshotting

---

## 📜 License

`arena-cache` is distributed under the terms of the MIT license. See [LICENSE](LICENSE) for details.
