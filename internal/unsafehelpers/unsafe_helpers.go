// Package unsafehelpers centralizes the use of the unsafe package in arena-cache.
// It provides helpers for zero-allocation conversions.
//
// ⚠️  **DISCLAIMER**   These helpers deliberately break the Go memory‑safety
// model for the sake of zero‑allocation conversions.  Use ONLY inside this
// repository; they are not part of the public API and may change without
// notice.  Misuse will lead to subtle data‑races or garbage‑collector
// corruption.
//
// All functions are `go:linkname`‑free, cgo‑free and pure Go 1.24.
//
// © 2025 arena-cache authors. MIT License.

package unsafehelpers

import "unsafe"

/* -------------------------------------------------------------------------
   1. Zero‑copy string/[]byte conversions
   ------------------------------------------------------------------------- */

// BytesToString converts a mutable byte slice to an immutable string without
// allocating.  The caller must guarantee that `b` will never be modified for
// the lifetime of the resulting string; otherwise the program exhibits
// undefined behaviour.
//
// Typical use‑case inside arena‑cache: hashing keys when K == []byte.
//
// DO NOT expose the returned string outside controlled scopes.
func BytesToString(b []byte) string {
    if len(b) == 0 {
        return ""
    }
    return unsafe.String(&b[0], len(b))
}

// StringToBytes re-interprets string data as a byte slice using unsafe.Pointer.
// The slice MUST remain read-only; writing to it will mutate immutable string storage and crash in future versions of Go.
func StringToBytes(s string) []byte {
    if len(s) == 0 {
        return nil
    }
    ptr := unsafe.StringData(s)
    return unsafe.Slice(ptr, len(s))
}

/* -------------------------------------------------------------------------
   2. Generic pointer → slice helpers
   ------------------------------------------------------------------------- */

// PtrSlice converts an arbitrary *T pointer + element count into a `[]T`
// without copying.  Useful when we need to treat an arena‑allocated array as a
// slice for iteration.  The slice is **still backed by arena memory** and thus
// safe from GC, but the usual rules about arena lifetime apply.
func PtrSlice[T any](ptr *T, n int) []T {
    if n == 0 {
        return nil
    }
    return unsafe.Slice(ptr, n)
}

// ByteSliceFrom returns a []byte view of raw memory starting at `ptr` with the
// given length.  Caller must ensure the memory block is at least `length`
// bytes.  Primarily used for hashing scalars where we only know the pointer
// and size at runtime.
func ByteSliceFrom(ptr unsafe.Pointer, length uintptr) []byte {
    return unsafe.Slice((*byte)(ptr), length)
}

/* -------------------------------------------------------------------------
   3. Alignment helpers
   ------------------------------------------------------------------------- */

// AlignUp rounds x up to the nearest multiple of align (which must be a power
// of two).  Fast bit‑twiddling alternative to math.Ceil for sizes.
func AlignUp(x, align uintptr) uintptr {
    return (x + align - 1) &^ (align - 1)
}

// IsPowerOfTwo returns true if x is a power of two (exactly one bit set).
func IsPowerOfTwo(x uintptr) bool {
    return x != 0 && (x&(x-1)) == 0
}
