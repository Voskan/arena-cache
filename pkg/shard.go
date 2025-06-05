package cache

import "context"

// shard.go contains the sharded segment of arena‑cache. A Cache is split into N
// independent shards to minimise lock contention.  Each shard keeps its own
// key‑>entry index, CLOCK‑Pro metadata ring and pointer to the current
// *generation (arena) it writes to.
//
// The code relies only on the standard library and the internal packages
// declared in this repository; there is **no cgo** and everything is safe for
// cross‑compilation.
//
// The shard is *not* exposed from the public API: all exported types live in
// pkg/cache.go.  Shards are created and managed by the top‑level Cache object.
//
// © 2025 arena‑cache authors. MIT License.

// shard owns all mutable structures for a slice of the key‑space.  Except for
// short critical sections protected by the RWMutex, all operations are
// lock‑free thanks to atomic primitives implemented in internal/clockpro.

// getOrLoad retrieves a value from the shard or loads it using the provided loader function.
func (s *shard[K, V]) getOrLoad(ctx context.Context, key K, loader LoaderFunc[K, V]) (V, error) {
    // Attempt to get the value from the shard
    if val, ok := s.get(key); ok {
        return val, nil
    }
    // Load the value using the loader function
    return loader(ctx, key)
}

// sizeBytes returns the total size in bytes of the shard.
func (s *shard[K, V]) sizeBytes() int64 {
    // Calculate the size based on the entries in the shard
    var total int64
    for _, entry := range s.index {
        total += int64(entry.weight)
    }
    return total
}

// close releases resources used by the shard.
func (s *shard[K, V]) close() {
    // Perform any necessary cleanup for the shard
    // For example, freeing arenas or clearing indices
    s.index = nil
    s.clock = nil
    s.genRing = nil
}
