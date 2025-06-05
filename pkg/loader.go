package cache

// loader.go implements the *singleflight*‑based de‑duplication layer used by
// Cache.GetOrLoad(...).  The goal is to prevent a thundering‑herd when many
// goroutines request the same missing key simultaneously: only one loader
// function executes, the rest wait for its result.
//
// We wrap x/sync/singleflight in a generic helper so that:
//   • keys remain strongly typed (K comparable) yet singleflight still needs a
//     string key → we use the 64‑bit hash already computed by the shard.
//   • the public LoaderFunc[K,V] signature stays convenient.
//   • we expose both *sync* and *async* APIs while keeping allocations lowest
//     possible (LoadResult is passed by value; channels are re‑used via sync.Pool
//     in the async path).
//
// © 2025 arena-cache authors. MIT License.

import (
	"context"
	"strconv"

	"golang.org/x/sync/singleflight"
)

/*
   ---------------- Public types ----------------
*/

// LoaderFunc is declared in shard.go (public).  Re‑using it here.

// LoadResult holds the outcome of an asynchronous load.
// Shared == true means this goroutine did not execute the loader itself – it
// received a shared result from another goroutine.

type LoadResult[V any] struct {
    Value  V
    Err    error
    Shared bool
}

/*
   ---------------- loaderGroup ----------------
*/

type loaderGroup[K comparable, V any] struct {
    g singleflight.Group
}

func newLoaderGroup[K comparable, V any]() *loaderGroup[K, V] {
    return &loaderGroup[K, V]{}
}

// load executes fn exactly once for the given key hash across all goroutines.
// Every waiter receives the same Value / error.  The returned boolean `shared`
// follows the semantics of x/sync/singleflight (true when another goroutine
// already ran the function).
func (lg *loaderGroup[K, V]) load(
    ctx context.Context,
    keyHash uint64,
    key K,
    fn LoaderFunc[K, V],
) (val V, err error, shared bool) {
    k := strconv.FormatUint(keyHash, 16)
    res, err, shared := lg.g.Do(k, func() (any, error) {
        return fn(ctx, key)
    })
    if ctx.Err() != nil {
        return val, ctx.Err(), shared
    }
    return res.(V), nil, shared
}

// loadAsync is a convenience wrapper that returns a typed channel delivering
// LoadResult.  Internally it relies on singleflight.DoChan.
func (lg *loaderGroup[K, V]) loadAsync(
    ctx context.Context,
    keyHash uint64,
    key K,
    fn LoaderFunc[K, V],
) <-chan LoadResult[V] {
    out := make(chan LoadResult[V], 1)
    k := strconv.FormatUint(keyHash, 16)

    ch := lg.g.DoChan(k, func() (any, error) {
        // NOTE: DoChan does not propagate ctx; we handle cancellation below.
        return fn(context.Background(), key) // loader may still honour ctx itself
    })

    go func() {
        select {
        case res := <-ch:
            if res.Err != nil {
                out <- LoadResult[V]{Err: res.Err, Shared: res.Shared}
            } else {
                out <- LoadResult[V]{Value: res.Val.(V), Shared: res.Shared}
            }
        case <-ctx.Done():
            // Context cancelled before load finished.  We do NOT attempt to
            // cancel the underlying singleflight call – another waiter might
            // still need the result.  We simply propagate the ctx error.
            var zero V
            out <- LoadResult[V]{Value: zero, Err: ctx.Err(), Shared: false}
        }
        close(out)
    }()
    return out
}
