package cache

// Package loaderfunc defines the LoaderFunc type used in arena-cache.
// It specifies the signature for cache loading functions.

// loaderfunc.go defines LoaderFunc – the user‑supplied callback that produces a
// value when Cache.GetOrLoad misses.  We place it in its own file so that it
// can be imported by multiple sub‑packages (cache.go, loader.go, etc.) without
// causing an import cycle.
//
// • The function must be **pure** and side‑effect free with regard to the
//   cache itself: it MUST NOT call Cache.Put or re‑enter the same Cache it
//   serves, otherwise deadlock or inconsistent state may occur.
// • It should honour the provided context for cancellation and deadlines.
// • If the loader returns an error, the value is not stored in the cache and
//   the error is propagated to the caller of GetOrLoad.
//
// K – key type, comparable (same as Cache).
// V – value type.
//
// © 2025 arena-cache authors. MIT License.

import "context"

// LoaderFunc is invoked by GetOrLoad when a key is absent. Implementations
// should return the value to cache or an error.  The same LoaderFunc instance
// may be invoked concurrently by different keys; it must therefore be
// thread‑safe.

type LoaderFunc[K comparable, V any] func(ctx context.Context, key K) (V, error)
