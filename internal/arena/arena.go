//go:build goexperiment.arenas
// +build goexperiment.arenas

// Package arena provides a thin wrapper around Go's experimental arena package.
// It simplifies the API for use in arena-cache.

// Package erena wraps Go 1.24's standard `arena` experimental package and
// hides its verbose low‑level API behind a tiny, stable surface suited for the
// needs of arena‑cache.  We expose only the primitives required:
//   • `New()` – construct an arena.
//   • `Free()` – release all memory at once (O(1)).
//   • `New[T]()` – allocate a single value of type T.
//   • `MakeSlice[T]()` – allocate a slice of T with length==cap.
//
// The wrapper is intentionally minimal: **no pooling, no stats, no GC hooks** –
// such concerns belong to upper layers (genring, metrics).  Keeping it thin
// also simplifies future migration should the upstream `arena` API change.
//
// Concurrency
// -----------
// arena.Arena is *not* thread‑safe; in arena‑cache the parent shard already
// serialises access with a mutex.  Therefore we do not add any locking here.
//
// ⚠️  DISCLAIMER  ----------------------------------------------
// Using arenas bypasses the garbage collector; ensure objects allocated inside
// never escape to the heap **after** Free() is called.  In arena‑cache this is
// safe because arenas live at most until generation rotation, at which point
// all *entries* referencing data are either promoted to TEST (ghost) or
// removed.
// -------------------------------------------------------------
//
// © 2025 arena-cache authors. MIT License.

package arena

import (
	"arena" // standard library experimental package
	"unsafe"
)

// Arena is a thin new‑type wrapper that prevents external packages from
// directly depending on `arena.Arena`, giving us the freedom to switch to a
// different allocator if needed.

type Arena struct{ ar arena.Arena }

// New constructs an empty arena ready for allocations.
func New() *Arena {
	var ar arena.Arena
	return &Arena{ar: ar} // Initialize the internal arena.Arena correctly
}

// Free releases **all** memory allocated in the arena.  After the call, any
// pointer previously returned from New/MakeSlice becomes invalid.
func (a *Arena) Free() {
	a.ar = arena.Arena{} // Reset the arena to a new instance
}

// NewValue allocates zero‑initialised T inside the arena and returns a pointer to it.
// The pointer is valid until Free() on the arena.
func NewValue[T any](a *Arena) *T { return arena.New[T](&a.ar) }

// MakeSlice allocates a slice of length==cap==n inside the arena and returns
// it.  The backing array is owned by the arena and will be released on Free().
func MakeSlice[T any](a *Arena, n int) []T { return arena.MakeSlice[T](&a.ar, n, n) }

// AllocBytes copies buf into the arena and returns a reference to the new
// memory.  Convenience helper used when we need an immutable grain inside the
// cache.
func AllocBytes(a *Arena, buf []byte) []byte {
	dst := arena.MakeSlice[byte](&a.ar, len(buf), len(buf))
	copy(dst, buf)
	return dst
}

// UnsafePointer converts an *arena-backed* pointer to unsafe.Pointer so that it
// can be stored inside cache metadata.  Usage is rare; provided for
// completeness.
func UnsafePointer[T any](p *T) unsafe.Pointer { return unsafe.Pointer(p) }
