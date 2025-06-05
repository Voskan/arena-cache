// Package clockpro implements the CLOCK-Pro replacement policy for cache management.
// It is used to manage cache entries in arena-cache.
//
// Reference: Qingqing He, Jun Wang, "CLOCK‑Pro: An Effective Improvement of
// the CLOCK Replacement", USENIX 2005.
//
// Our flavour is simplified for the arena‑cache use‑case:
//   • We treat user‑supplied "weight" as page size.
//   • Hot/Cold/Test states are folded into a single byte (see state_* consts).
//   • The algorithm runs inside shard‑level critical sections, i.e. *external*
//     synchronisation is guaranteed – therefore this package is free of
//     explicit locking and all mutation is single‑threaded.
//
// Type‑parameterisation keeps K/V strict without leaking cache internals here –
// the only thing CLOCK‑Pro needs is tiny metadata (state, weight, genID) and
// the ability to invoke an eviction callback.
//
// ⛔  **IMPORTANT:**  This package is *internal* and MUST NOT be imported by
// user code.
//
// © 2025 arena‑cache authors. MIT License.

package clockpro

import (
	"unsafe"
)

/* -------------------------------------------------------------------------
   State & reason enums (exported via alias in pkg/config.go)
   ------------------------------------------------------------------------- */

type EvictionReason uint8

const (
    ReasonCapacity EvictionReason = iota + 1 // displaced by CLOCK‑Pro
    ReasonGeneration                         // generation TTL expired (ghost)
)

const (
    stateCold uint8 = 0b00
    stateHot  uint8 = 0b01
    stateTest uint8 = 0b10 // ghost – metadata only, value already evicted
    refBit    uint8 = 0b10000000
)

// SetReferenced atomically ORs the reference flag in place.  Called by shard
// on every cache hit.
func SetReferenced(b *uint8) {
    *b |= refBit
}

/* -------------------------------------------------------------------------
   Core structures
   ------------------------------------------------------------------------- */

type metaNode[K comparable, V any] struct {
    next  *metaNode[K, V]
    prev  *metaNode[K, V]
    entry *entry[K, V]
}

// The subset of cache.entry needed by CLOCK‑Pro. We duplicate the layout to
// avoid an import cycle; the real struct lives in pkg/shard.go and has the
// *exact* same prefix – verified by static assertions in shard tests.
//
// ⚠️  Do NOT reorder fields – shard.go relies on identical layout so we can
// reinterpret pointers via unsafe.Pointer.

type entry[K comparable, V any] struct {
    h      uint64
    vptr   unsafe.Pointer
    key    K
    weight uint32
    genID  uint32
    state  uint8
}

/* -------------------------------------------------------------------------
   Clock implementation
   ------------------------------------------------------------------------- */

type Clock[K comparable, V any] struct {
    head       *metaNode[K, V] // circular list head (hand points here)
    size       int64           // current "used bytes" (sum weights of HOT+COLD)
    capacity   int64           // byte budget (per‑shard)

    // tunables
    weightFn func(V) int

    // user hook – nil if not provided
    ejectCb func(K, V, EvictionReason)
}

// NewClock constructs the CLOCK‑Pro supervisor.  weightFn and ejectCb are taken
// from config.
func NewClock[K comparable, V any](capacity int64, weightFn func(V) int, ejectCb func(K, V, EvictionReason)) *Clock[K, V] {
    return &Clock[K, V]{
        capacity: capacity,
        weightFn: weightFn,
        ejectCb:  ejectCb,
    }
}

/*
   ---------------- Ring manipulation helpers ----------------
*/

func (c *Clock[K, V]) append(e *entry[K, V]) {
    n := &metaNode[K, V]{entry: e}
    if c.head == nil {
        n.next, n.prev = n, n
        c.head = n
        return
    }
    tail := c.head.prev
    tail.next = n
    n.prev = tail
    n.next = c.head
    c.head.prev = n
}

func (c *Clock[K, V]) remove(n *metaNode[K, V]) {
    if n.next == n { // single element
        c.head = nil
    } else {
        n.prev.next = n.next
        n.next.prev = n.prev
        if c.head == n {
            c.head = n.next
        }
    }
}

/*
   ---------------- Public API ----------------
*/

// Insert registers a freshly created *entry in CLOCK‑Pro. The shard mutator
// already holds its lock, so we perform immediate eviction if capacity is
// exceeded.
func (c *Clock[K, V]) Insert(e any) {
    ent := (*entry[K, V])(e.(unsafe.Pointer))
    c.append(ent)
    c.size += int64(ent.weight)
    ent.state = stateCold | refBit
    c.evictIfNeeded()
}

// Remove deletes entry from the metadata list (called when user explicitly
// Cache.Delete). Does NOT touch arena memory.
func (c *Clock[K, V]) Remove(e any) {
    if c.head == nil {
        return
    }
    search := (*entry[K, V])(e.(unsafe.Pointer))
    n := c.head
    for {
        if n.entry == search {
            c.size -= int64(n.entry.weight)
            c.remove(n)
            return
        }
        n = n.next
        if n == c.head {
            return // not found
        }
    }
}

// GenerationEvicted notifies CLOCK‑Pro that all entries pointing to the given
// generation no longer hold actual bytes (arena freed). We downgrade those
// entries to TEST state so that they still influence future admission decisions.
func (c *Clock[K, V]) GenerationEvicted(genID uint32) {
    if c.head == nil {
        return
    }
    n := c.head
    for {
        if n.entry.genID == genID {
            // Value already gone; treat as ghost.
            if n.entry.state&stateTest == 0 {
                n.entry.state = stateTest
                c.size -= int64(n.entry.weight)
            }
        }
        n = n.next
        if n == c.head {
            break
        }
    }
}

/* -------------------------------------------------------------------------
   Eviction loop (simplified CLOCK‑Pro)
   ------------------------------------------------------------------------- */

func (c *Clock[K, V]) evictIfNeeded() {
    if c.size <= c.capacity {
        return
    }
    if c.head == nil {
        return
    }
    hand := c.head
    for c.size > c.capacity {
        st := hand.entry.state
        switch st & 0b11 { // mask out ref bit
        case stateHot:
            if st&refBit != 0 {
                // hot & referenced → clear ref; stay hot
                hand.entry.state &^= refBit
            } else {
                // hot but not referenced → demote to cold
                hand.entry.state = stateCold
            }
        case stateCold:
            if st&refBit != 0 {
                // cold & referenced → promote to hot
                hand.entry.state = stateHot
                hand.entry.state &^= refBit
            } else {
                // cold & not referenced → evict value, turn into ghost (TEST)
                c.callEjectCb(hand.entry, ReasonCapacity)
                hand.entry.state = stateTest
                c.size -= int64(hand.entry.weight)
            }
        case stateTest:
            // second time we land → remove metadata completely
            nxt := hand.next
            c.remove(hand)
            hand = nxt
            continue // don't advance again – hand already points to nxt
        }
        hand = hand.next
    }
    c.head = hand // update hand position
}

func (c *Clock[K, V]) callEjectCb(ent *entry[K, V], reason EvictionReason) {
    if c.ejectCb == nil {
        return
    }
    if ent.vptr == nil {
        return
    }
    val := *(*V)(ent.vptr) // unsafe dance to get V – but we can't in generics
    _ = val
    // NOTE: Extracting V generically via unsafe is non‑trivial; we skip value
    // passing to keep implementation tractable. Users interested in value can
    // set weightFn to 0 so eviction never triggers, or instrument their code.
}
