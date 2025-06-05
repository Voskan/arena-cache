package cache

// Package config contains configuration options for arena-cache.
// It defines default settings and allows customization.

// config.go defines the internal configuration object and the set of
// functional options that can be passed to New[K,V].  A generic Option is used
// so that callbacks retain full type‑safety with respect to the concrete value
// type V and key type K chosen by the user.
//
// Design notes
// ------------
// • All fields are initialised with sensible defaults in defaultConfig().
// • Options never allocate unless strictly necessary – they just capture
//   pointers to external objects (registry, logger …).
// • We hide the struct from public API: users can only influence behaviour via
//   Option[K,V].  This guarantees forward compatibility.
//
// © 2025 arena-cache authors. MIT License.

import (
	"time"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"errors"

	"github.com/Voskan/arena-cache/internal/clockpro"
)

// WeightFn calculates an integer weight for the stored value V. The number is
// abstract – the eviction algorithm treats it as *relative* cost (e.g. bytes,
// points, whatever makes sense for the application).  Must always return a
// positive number, otherwise the value is treated as weight=1.
// The function MUST be pure: side‑effects are not allowed.
//
// Implementers should make the function as cheap as possible – it runs on every
// Put() call.

type WeightFn[V any] func(V) int

// EjectCallback is invoked when an item is evicted (TTL expiration is NOT
// considered an eviction – callback is only for capacity based CLOCK‑Pro
// decisions). The reason enum comes from the internal clockpro package but is
// exported through Option for convenience.

//go:export
// To avoid leaking internal package path to users we re‑export the type via a
// type alias.

type EjectReason = clockpro.EvictionReason

type EjectCallback[K comparable, V any] func(key K, val V, reason EjectReason)

// Option is the functional option passed to New.  It is generic because some
// options (WeightFn, EjectCallback) refer to concrete K/V types.

type Option[K comparable, V any] func(*config[K, V])

// config bundles every knob that influences cache behaviour.  All fields are
// immutable once the Cache is constructed – we do not support live mutation
// from user land; hot‑reload of TTL etc. would complicate correctness proofs.

type config[K comparable, V any] struct {
    // memory & shards are copied from the New() arguments; kept here just for
    // completeness so that all params live in one object.
    capBytes int64
    ttl      time.Duration
    shards   uint8

    // optional knobs
    registry  *prometheus.Registry
    logger    *zap.Logger
    weightFn  WeightFn[V]
    ejectCb   EjectCallback[K, V]
    partID    int // reserved for future partition‑pinning feature

    // derived / pre‑computed values – filled in finalise().
    rotationStep time.Duration
}

/*
   ---------------- Default configuration ----------------
*/

func defaultWeightFn[V any](v V) int {
    w := int(unsafe.Sizeof(v))
    if w <= 0 {
        return 1
    }
    return w
}

func defaultConfig[K comparable, V any](capBytes int64, ttl time.Duration, shards uint8) *config[K, V] {
    return &config[K, V]{
        capBytes: capBytes,
        ttl:      ttl,
        shards:   shards,
        weightFn: defaultWeightFn[V],
        logger:   zap.NewNop(),
        registry: nil, // user must opt‑in to metrics
    }
}

/*
   ---------------- Functional options exposed to users ----------------
*/

// WithMetrics enables Prometheus metrics collection for the cache instance.
// Passing nil disables metrics (default).
func WithMetrics[K comparable, V any](reg *prometheus.Registry) Option[K, V] {
    return func(c *config[K, V]) {
        c.registry = reg
    }
}

// WithLogger plugs an external zap.Logger.  The cache never logs on the hot
// path; only slow events (arena rotation, severe errors) are emitted.
func WithLogger[K comparable, V any](l *zap.Logger) Option[K, V] {
    return func(c *config[K, V]) {
        if l != nil {
            c.logger = l
        }
    }
}

// WithWeightFn overrides the default size‑based weight calculation.
// The provided function must be cheap and deterministic.
func WithWeightFn[K comparable, V any](fn WeightFn[V]) Option[K, V] {
    return func(c *config[K, V]) {
        if fn != nil {
            c.weightFn = fn
        }
    }
}

// WithEjectCallback registers a function that will be invoked whenever an item
// is evicted due to capacity pressure (CLOCK‑Pro).  The callback runs in the
// calling goroutine and **must not block** – otherwise overall latency will
// suffer. Heavy IO should be deferred to another goroutine.
func WithEjectCallback[K comparable, V any](cb EjectCallback[K, V]) Option[K, V] {
    return func(c *config[K, V]) {
        c.ejectCb = cb
    }
}

// Reserved for future public API – partition pinning.
// func WithPartition[K comparable, V any](id int) Option[K, V] { … }

/*
   ---------------- Helper: apply options & validate ----------------
*/

// applyOptions copies user‑supplied options into cfg, validates invariants and
// pre‑computes rotationStep.
func applyOptions[K comparable, V any](cfg *config[K, V], opts []Option[K, V]) error {
    for _, opt := range opts {
        opt(cfg)
    }

    // Validation – bail out early with descriptive error.
    if cfg.capBytes <= 0 {
        return errInvalidCap
    }
    if cfg.ttl <= 0 {
        return errInvalidTTL
    }
    if cfg.shards == 0 || (cfg.shards&(cfg.shards-1)) != 0 {
        return errInvalidShards
    }

    // Derive rotation step: we want at least two generations to coexist, so we
    // split TTL into (#gens) slots where #gens = ceil(capBytes / avgArenaSize).
    // For now we assume 4 generations; in future we might autotune this.
    const generations = 4
    cfg.rotationStep = cfg.ttl / generations
    if cfg.rotationStep < time.Millisecond {
        cfg.rotationStep = time.Millisecond
    }
    return nil
}

/*
   ---------------- Error values ----------------
*/

var (
    errInvalidCap    = errors.New("capacity bytes must be > 0")
    errInvalidTTL    = errors.New("ttl must be > 0")
    errInvalidShards = errors.New("shards must be power‑of‑two and > 0")
)
