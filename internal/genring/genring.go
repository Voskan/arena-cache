// Package genring maintains a circular buffer ("ring") of *generations* –
// time‑bounded arenas used by arena‑cache to implement O(1) TTL expiration and
// bulk memory release.
//
// A *generation* owns:
//   • an arena (outside GC‑heap) where values are allocated;
//   • accounting of bytes (approximate – user weightFn provides the numbers);
//   • creation timestamp;
//   • monotonically increasing ID (uint32) so CLOCK‑Pro can track ghosts after
//     the arena itself has been freed.
//
// Concurrency model
// -----------------
// genring does **not** use its own locks – the parent shard already serialises
// access with its mutex.  All exported methods therefore assume external
// synchronisation except where atomic is explicitly used (bytes counters).
//
// © 2025 arena-cache authors. MIT License.

package genring

import (
	"sync/atomic"
	"time"

	arena "github.com/Voskan/arena-cache/internal/arena"
)

/* -------------------------------------------------------------------------
   Generation object
   ------------------------------------------------------------------------- */

type generation struct {
    id      uint32
    ar      *arena.Arena // nil once freed
    created time.Time
    bytes   atomic.Int64 // live bytes recorded via weightFn() heuristic
}

func newGeneration(id uint32) *generation {
    gen := &generation{
        id:      id,
        ar:      arena.New(),
        created: time.Now(),
    }
    if gen.ar == nil {
        panic("Arena is nil after initialization")
    }
    return gen
}

// ID returns the stable identifier for the generation.
func (g *generation) ID() uint32 { return g.id }

// Arena exposes the underlying arena for allocation.  It is valid until the
// generation is rotated out and g.ar becomes nil.
func (g *generation) Arena() *arena.Arena { return g.ar }

// addBytes increments the accounting counter; used by Ring.CheckRotationNeeded.
func (g *generation) addBytes(n int64) { g.bytes.Add(n) }

// size returns bytes currently attributed to this generation.
func (g *generation) size() int64 { return g.bytes.Load() }

// free releases the arena memory and prepares the object to act as a ghost –
// the id remains, but allocations must no longer target this generation.
func (g *generation) free() {
    if g.ar != nil {
        g.ar.Free()
        g.ar = nil
    }
}

/* -------------------------------------------------------------------------
   Ring – public API used by shard
   ------------------------------------------------------------------------- */

type Ring[K comparable, V any] struct {
    gens        []*generation
    activeIdx   int
    ttl         time.Duration
    perGenBytes int64

    idCtr atomic.Uint32
}

const defaultGenerations = 4 // may be tuned in future

// New constructs a generation ring sized for the given capacity and TTL.
// capBytes is capacity *per shard*.
func New[K comparable, V any](capBytes int64, ttl time.Duration) *Ring[K, V] {
    if capBytes <= 0 {
        panic("genring: capBytes must be positive")
    }
    if ttl <= 0 {
        panic("genring: ttl must be positive")
    }

    r := &Ring[K, V]{
        ttl:         ttl,
        perGenBytes: capBytes / defaultGenerations,
    }
    if r.perGenBytes == 0 {
        r.perGenBytes = capBytes // tiny caches → single-gen capacity control
    }
    r.gens = make([]*generation, defaultGenerations)

    // Generation IDs start at 1 (0 reserved for "nil").
    r.idCtr.Store(1)
    first := newGeneration(r.idCtr.Load())
    r.gens[0] = first
    r.activeIdx = 0
    return r
}

// Active returns the generation currently used for new allocations.
func (r *Ring[K, V]) Active() *generation {
    return r.gens[r.activeIdx]
}

// CheckRotationNeeded is called on every Put. It adds delta bytes to the active
// generation and returns true if byte budget has been exceeded.
func (r *Ring[K, V]) CheckRotationNeeded(delta int64) bool {
    g := r.Active()
    g.addBytes(delta)
    return g.size() > r.perGenBytes
}

// Rotate advances the ring, creates a fresh generation, and frees the arena of
// whichever generation falls out of the TTL window.  The *freed* generation is
// returned so that CLOCK-Pro can retain its ghost metadata.  Returned pointer
// may be nil when the slot was empty (only happens before the ring is fully
// warmed up).
func (r *Ring[K, V]) Rotate() *generation {
    nextIdx := (r.activeIdx + 1) % len(r.gens)

    // Free the arena of the generation we are about to overwrite.
    dead := r.gens[nextIdx]
    if dead != nil {
        dead.free()
    }

    // Allocate fresh generation into the slot.
    newID := r.idCtr.Add(1)
    fresh := newGeneration(newID)
    r.gens[nextIdx] = fresh
    r.activeIdx = nextIdx
    return dead
}

// LiveBytes sums approximate sizes across all generations.  Cheap enough for
// sporadic calls.
func (r *Ring[K, V]) LiveBytes() int64 {
    var total int64
    for _, g := range r.gens {
        if g != nil {
            total += g.size()
        }
    }
    return total
}
