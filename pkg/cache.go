package cache

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

import (
	"context"
	"errors"
	"hash/maphash"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	arena "github.com/Voskan/arena-cache/internal/arena"
	"github.com/Voskan/arena-cache/internal/clockpro"
	"github.com/Voskan/arena-cache/internal/genring"
)

// entryState encodes CLOCK‑Pro flags in a compact form.
const (
    stateCold   uint8 = 0b00 // item is cold and not recently referenced
    stateHot    uint8 = 0b01 // item is hot (frequently used)
    stateTest   uint8 = 0b10 // ghost entry – remembered after eviction
    refBit      uint8 = 0b10000000 // high bit is the *reference* (R) flag
)

// entry is the metadata kept for every cached item. It purposefully fits into
// 48 bytes on 64‑bit architectures to maximise cache‑line utilisation:
//   • 8 B – hashed key (for fast compare)
//   • 8 B – unsafe pointer to the value inside arena
//   • 16 B – key (when K is string interface header) or zero‑sized field (when
//            K is scalar); we store the real key only to be able to call the
//            user‑supplied eviction callback.
//   • 4 B – weight
//   • 4 B – generation ID
//   • 1 B – state/ref bits
//   • 3 B – padding/alignment
// NB: exact size varies with K, yet the struct is cache‑line‑friendly.
//
// We keep the *arena pointer* instead of value directly to guarantee that the
// Go GC never scans the object graph stored in the cache – the arena resides
// outside of the managed heap.

type entry[K comparable, V any] struct {
    h       uint64         // SipHash‑64 of the key (pre‑computed)
    vptr    unsafe.Pointer // *V allocated inside arena
    key     K              // original key (for callbacks, Delete)
    weight  uint32         // user‑defined weight units
    genID   uint32         // generation that owns this value
    state   uint8          // CLOCK‑Pro state + R‑bit
}

// shard owns all mutable structures for a slice of the key‑space.  Except for
// short critical sections protected by the RWMutex, all operations are
// lock‑free thanks to atomic primitives implemented in internal/clockpro.

type shard[K comparable, V any] struct {
    mu   sync.RWMutex

    // index maps hashed key → *entry. We keep the real key inside the entry so
    // hash collisions are handled with a single key comparison.
    index map[uint64]*entry[K, V]

    // clock keeps CLOCK‑Pro metadata.  It is implemented in its own package to
    // keep this file focused on high‑level cache behaviour.
    clock *clockpro.Clock[K, V]

    // generations ring provides arenas for allocation and takes care of TTL &
    // capacity based rotation.
    genRing *genring.Ring[K, V]

    // stats – fast counters for Prometheus (atomic to avoid locking on hot
    // path).
    hits      atomic.Uint64
    misses    atomic.Uint64
    evictions atomic.Uint64

    // hash seed – each shard owns its own maphash.Seed to avoid global locks.
    seed maphash.Seed
}

// newShard constructs an empty shard. It assumes the caller already validated
// all arguments (capBytes > 0, ttl > 0, etc.)
func newShard[K comparable, V any](capBytes int64, ttl time.Duration, weightFn func(V) int,
    ejectCb func(K, V, clockpro.EvictionReason),
) *shard[K, V] {
    s := &shard[K, V]{
        index:  make(map[uint64]*entry[K, V], 1024), // start with 1k slots
        clock:  clockpro.NewClock[K, V](capBytes, weightFn, ejectCb),
        genRing: genring.New[K, V](capBytes, ttl),
        seed:   maphash.MakeSeed(),
    }
    return s
}

// hash returns SipHash‑64 of the provided key using shard‑local seed.
func (s *shard[K, V]) hash(key K) uint64 {
    var h maphash.Hash
    h.SetSeed(s.seed)
    // Use type switch to avoid reflection for common key types.
    switch k := any(key).(type) {
    case string:
        h.WriteString(k)
    case []byte:
        h.Write(k)
    default:
        // For scalars we rely on unsafe – convert address to slice of bytes.
        // This is safe because we only use the value for hashing.
        ptr := unsafe.Pointer(&key)
        size := unsafe.Sizeof(key)
        bytes := unsafe.Slice((*byte)(ptr), size)
        h.Write(bytes)
    }
    return h.Sum64()
}

/*
   -------- Public‑facing methods called by the parent Cache --------
*/

// get returns the value pointer (residing in arena) and a flag whether the item
// was found.  It updates CLOCK‑Pro metadata in lock‑free manner.
func (s *shard[K, V]) get(key K) (val V, ok bool) {
    h := s.hash(key)

    s.mu.RLock()
    ent, found := s.index[h]
    s.mu.RUnlock()

    if !found {
        s.misses.Add(1)
        return val, false
    }

    // Ensure the keys are equal in case of hash collision.
    if ent.key != key {
        s.misses.Add(1)
        return val, false
    }

    s.hits.Add(1)
    // Mark as referenced for CLOCK‑Pro algorithm.
    clockpro.SetReferenced(&ent.state)

    // Dereference unsafe pointer – the value lives inside arena, we must copy
    // it for the caller (it will escape to heap but that's caller's choice).
    vp := (*V)(ent.vptr)
    if vp == nil {
        // Should never happen: invariant violation.
        return val, false
    }
    return *vp, true
}

// put inserts or updates a value.  The upsert path is GC‑free: the new value is
// allocated inside the *current* generation arena; metadata is written, and if
// there is an older entry — it stays intact until its generation rotates out.
//
// weight allows the caller to express relative cost (bytes, logical weight…).
func (s *shard[K, V]) put(key K, val V, weight int) {
    h := s.hash(key)

    // Fast path: optimistic read‑lock, upgrade on miss.
    s.mu.RLock()
    if old, ok := s.index[h]; ok && old.key == key {
        // Update hot path – no need for hash collision check twice.
        // We merely overwrite the value pointer & weight; key remains.
        gen := s.genRing.Active()
        ptr := arena.NewValue[V](gen.Arena())
        *ptr = val

        old.vptr = unsafe.Pointer(ptr)
        atomic.StoreUint32(&old.weight, uint32(weight))
        old.genID = gen.ID()
        s.mu.RUnlock()
        return
    }
    s.mu.RUnlock()

    // Slow path: need exclusive lock to insert a fresh entry.
    s.mu.Lock()
    defer s.mu.Unlock()

    gen := s.genRing.Active()
    ptr := arena.NewValue[V](gen.Arena())
    *ptr = val

    ent := &entry[K, V]{
        h:      h,
        vptr:   unsafe.Pointer(ptr),
        key:    key,
        weight: uint32(weight),
        genID:  gen.ID(),
        state:  stateCold | refBit, // new entry is cold but referenced
    }

    // Insert into primary index.
    s.index[h] = ent

    // Register in CLOCK‑Pro (may trigger internal eviction).
    s.clock.Insert(ent)

    // If generation grew beyond capacity – rotate.
    if s.genRing.CheckRotationNeeded(int64(weight)) {
        s.rotate()
    }
}

// delete removes key from the shard. It does not free the underlying arena
// memory immediately (that happens on generation rotation).
func (s *shard[K, V]) delete(key K) {
    h := s.hash(key)

    s.mu.Lock()
    ent, ok := s.index[h]
    if ok && ent.key == key {
        delete(s.index, h)
        s.clock.Remove(ent)
        s.evictions.Add(1)
    }
    s.mu.Unlock()
}

// rotate is called by the parent Cache at a scheduled interval or when the
// active generation exceeds its byte budget.  All logic is delegated to
// genRing, while CLOCK‑Pro is notified about the new generation so that ghost
// entries from freed arenas may still influence replacement policy.
func (s *shard[K, V]) rotate() {
    deadGen := s.genRing.Rotate()
    if deadGen == nil {
        return // nothing to free yet
    }

    // Inform CLOCK‑Pro – ghost entries are kept with stateTest so they survive
    // for a while and affect admission decisions.
    s.clock.GenerationEvicted(deadGen.ID())
}

// len returns *approximate* number of live items (RLock used – safe for hot
// path, may slightly undercount during rotation).
func (s *shard[K, V]) len() int {
    s.mu.RLock()
    n := len(s.index)
    s.mu.RUnlock()
    return n
}

// statsSnapshot returns atomic counters – useful for prometheus scraping.
func (s *shard[K, V]) statsSnapshot() (hits, misses, evict uint64) {
    return s.hits.Load(), s.misses.Load(), s.evictions.Load()
}

// Cache represents the main cache structure.
type Cache[K comparable, V any] struct {
    shards []*shard[K, V]
}

// New creates a new cache instance with the specified capacity, TTL, and shard count.
func New[K comparable, V any](capBytes int64, ttl time.Duration, shards uint8, opts ...Option[K, V]) (*Cache[K, V], error) {
    // Validate input parameters
    if capBytes <= 0 {
        return nil, errors.New("capacity bytes must be > 0")
    }
    if ttl <= 0 {
        return nil, errors.New("ttl must be > 0")
    }
    if shards == 0 || (shards&(shards-1)) != 0 {
        return nil, errors.New("shards must be power-of-two and > 0")
    }

    // Create default configuration
    cfg := defaultConfig[K, V](capBytes, ttl, shards)

    // Apply options
    if err := applyOptions(cfg, opts); err != nil {
        return nil, err
    }

    // Initialize cache
    c := &Cache[K, V]{
        shards: make([]*shard[K, V], shards),
    }
    for i := range c.shards {
        c.shards[i] = newShard(capBytes/int64(shards), ttl, cfg.weightFn, cfg.ejectCb)
    }

    return c, nil
}

// Put inserts a value into the cache.
func (c *Cache[K, V]) Put(ctx context.Context, key K, value V, weight int) {
    shard := c.shards[c.shardIndex(key)]
    shard.put(key, value, weight)
}

// GetOrLoad retrieves a value from the cache or loads it using the provided loader function.
func (c *Cache[K, V]) GetOrLoad(ctx context.Context, key K, loader LoaderFunc[K, V]) (V, error) {
    shard := c.shards[c.shardIndex(key)]
    return shard.getOrLoad(ctx, key, loader)
}

// Len returns the total number of items in the cache.
func (c *Cache[K, V]) Len() int {
    total := 0
    for _, shard := range c.shards {
        total += shard.len()
    }
    return total
}

// SizeBytes returns the total size in bytes of the cache.
func (c *Cache[K, V]) SizeBytes() int64 {
    total := int64(0)
    for _, shard := range c.shards {
        total += shard.sizeBytes()
    }
    return total
}

// shardIndex calculates the index of the shard for a given key.
func (c *Cache[K, V]) shardIndex(key K) int {
    return int(c.shards[0].hash(key) % uint64(len(c.shards)))
}

// Close releases resources used by the cache.
func (c *Cache[K, V]) Close() {
    for _, shard := range c.shards {
        shard.close()
    }
}
