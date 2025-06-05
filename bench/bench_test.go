// Package bench provides reproducible micro‑benchmarks for arena‑cache.
// Run via:  go test ./bench -bench=. -benchmem -cpu 1,4,16
//
// The benchmarks intentionally use a *single* key/value shape so results are
// comparable across versions:
//   • Key   – uint64  (cheap hashing, fits in register)
//   • Value – 64‑byte struct (large enough to matter, small enough for cache)
//
// We measure:
//   1. Put          – write‑only workload
//   2. Get          – read‑only workload (after warm‑up)
//   3. GetParallel  – highly concurrent reads (b.RunParallel)
//   4. GetOrLoad    – 90% hits, 10% misses with loader cost
//
// Results are printed in ns/op + alloc/op so CI can diff via benchstat.
//
// NOTE: Unit tests live elsewhere; this file is *only* for performance.
//
// © 2025 arena‑cache authors. MIT License.

package bench

import (
	"context"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	cache "github.com/Voskan/arena-cache/pkg"
)

/* -------------------------------------------------------------------------
   Test harness helpers
   ------------------------------------------------------------------------- */

type value64 struct {
    _ [64]byte
}

const (
    capBytes = 64 << 20 // 64 MiB per shard cap
    ttl       = time.Minute
    shards    = 16
    keys      = 1 << 20 // 1M keys for dataset
)

func newTestCache() *cache.Cache[uint64, value64] {
    c, err := cache.New[uint64, value64](capBytes, ttl, shards)
    if err != nil {
        panic(err)
    }
    return c
}

// global dataset reused across benches to avoid reallocating large slices.
var ds = func() []uint64 {
    arr := make([]uint64, keys)
    for i := range arr {
        arr[i] = rand.Uint64()
    }
    return arr
}()

/* -------------------------------------------------------------------------
   Benchmarks
   ------------------------------------------------------------------------- */

func BenchmarkPut(b *testing.B) {
    c := newTestCache()
    val := value64{}
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        key := ds[i& (keys-1)]
        c.Put(context.Background(), key, val, 1)
    }
    c.Close()
}

func BenchmarkGet(b *testing.B) {
    c := newTestCache()
    val := value64{}
    // pre‑populate (warm‑up)
    for _, k := range ds {
        c.Put(context.Background(), k, val, 1)
    }
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        k := ds[i & (keys-1)]
        _, _ = c.GetOrLoad(context.Background(), k, func(ctx context.Context, key uint64) (value64, error) {
            return val, nil
        })
    }
    c.Close()
}

func BenchmarkGetParallel(b *testing.B) {
    c := newTestCache()
    val := value64{}
    for _, k := range ds {
        c.Put(context.Background(), k, val, 1)
    }
    b.ReportAllocs()
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        idx := rand.Intn(keys)
        for pb.Next() {
            idx = (idx + 1) & (keys - 1)
            c.GetOrLoad(context.Background(), ds[idx], func(ctx context.Context, key uint64) (value64, error) {
                return val, nil
            })
        }
    })
    c.Close()
}

func BenchmarkGetOrLoad(b *testing.B) {
    c := newTestCache()
    val := value64{}
    // Preload 90% of keys to simulate mixed hit/miss.
    for i, k := range ds {
        if i%10 != 0 { // 90% fill
            c.Put(context.Background(), k, val, 1)
        }
    }
    var loaderCnt atomic.Uint64
    loader := func(ctx context.Context, key uint64) (value64, error) {
        loaderCnt.Add(1)
        return val, nil
    }
    b.ReportAllocs()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        k := ds[i&(keys-1)]
        c.GetOrLoad(context.Background(), k, loader)
    }
    c.Close()
    b.ReportMetric(float64(loaderCnt.Load())/float64(b.N)*100, "miss-%")
}

/* -------------------------------------------------------------------------
   Utility – ensure deterministic Rand for repeatability
   ------------------------------------------------------------------------- */

func init() {
    rand.Seed(42)
    runtime.GOMAXPROCS(runtime.NumCPU())
}
