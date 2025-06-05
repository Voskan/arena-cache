package cache

// metrics.go contains a thin abstraction over Prometheus so that arena-cache
// can be used with or without metrics.  When the user passes a *prometheus.Registry
// in New(..., WithMetrics(reg)), we create labeled metrics and expose them via
// the registry.  Otherwise a no‑op sink is used and the hot‑path does not pay
// for metric updates.
//
// All metrics are **shard‑level**; aggregations can easily be done on the
// Prometheus side via sum() / rate().  We keep the implementation minimal to
// avoid a hard dependency on any particular monitoring stack.
//
// Metric names follow Prometheus best practices, suffixed with "_total" for
// counters.  The `arena_bytes` gauge reflects live arena memory per shard.
//
// ┌─────────────────────────────────────┐
// │ Metric              │ Type │ Labels │
// ├──────────────────────┼──────┼────────┤
// │ cache_hits_total     │ Ctr  │ shard  │
// │ cache_misses_total   │ Ctr  │ shard  │
// │ cache_evictions_total│ Ctr  │ shard  │
// │ arena_rotations_total│ Ctr  │ shard  │
// │ arena_bytes          │ Gge  │ shard  │
// └─────────────────────────────────────┘
//
// © 2025 arena-cache authors. MIT License.

import (
	"strconv"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

/*
   ---------------- Public (package‑level) API ----------------
*/

// metricsSink is an internal interface abstracting away the concrete backend
// (Prometheus vs noop).  It is *not* exposed outside the package; Cache and
// shards only know about the generic methods here.

type metricsSink interface {
    incHit(shard uint8)
    incMiss(shard uint8)
    incEvict(shard uint8)
    incRotation(shard uint8)
    addArenaBytes(shard uint8, delta int64)
    setArenaBytes(shard uint8, value int64)
}

/*
   ---------------- No‑op implementation ----------------
*/

type noopMetrics struct{}

func (noopMetrics) incHit(uint8)                 {}
func (noopMetrics) incMiss(uint8)                {}
func (noopMetrics) incEvict(uint8)               {}
func (noopMetrics) incRotation(uint8)            {}
func (noopMetrics) addArenaBytes(uint8, int64)   {}
func (noopMetrics) setArenaBytes(uint8, int64)   {}

/*
   ---------------- Prometheus implementation ----------------
*/

type promMetrics struct {
    hits      *prometheus.CounterVec
    misses    *prometheus.CounterVec
    evictions *prometheus.CounterVec
    rotations *prometheus.CounterVec
    arena     *prometheus.GaugeVec

    // For arenas we also keep atomic mirrors so that Rotator can compute delta
    // without calling WithLabelValues() on the hot path.
    arenaMirror []atomic.Int64 // len == shardCount
}

func newPromMetrics(shardCount int, reg *prometheus.Registry) *promMetrics {
    label := []string{"shard"}

    pm := &promMetrics{
        hits: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "arena_cache",
                Name:      "hits_total",
                Help:      "Number of cache hits.",
            }, label),
        misses: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "arena_cache",
                Name:      "misses_total",
                Help:      "Number of cache misses.",
            }, label),
        evictions: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "arena_cache",
                Name:      "evictions_total",
                Help:      "Number of items evicted by CLOCK-Pro.",
            }, label),
        rotations: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "arena_cache",
                Name:      "arena_rotations_total",
                Help:      "Number of arena rotations (TTL or capacity).",
            }, label),
        arena: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: "arena_cache",
                Name:      "arena_bytes",
                Help:      "Live bytes allocated in arenas.",
            }, label),
        arenaMirror: make([]atomic.Int64, shardCount),
    }

    // Register collectors. If registry is nil the caller decided to disable
    // metrics; function should never be called with nil.
    reg.MustRegister(pm.hits, pm.misses, pm.evictions, pm.rotations, pm.arena)
    return pm
}

/*
   -------- promMetrics implements metricsSink --------
*/

func (m *promMetrics) incHit(shard uint8) {
    m.hits.WithLabelValues(strconv.Itoa(int(shard))).Inc()
}
func (m *promMetrics) incMiss(shard uint8) {
    m.misses.WithLabelValues(strconv.Itoa(int(shard))).Inc()
}
func (m *promMetrics) incEvict(shard uint8) {
    m.evictions.WithLabelValues(strconv.Itoa(int(shard))).Inc()
}
func (m *promMetrics) incRotation(shard uint8) {
    m.rotations.WithLabelValues(strconv.Itoa(int(shard))).Inc()
}
func (m *promMetrics) addArenaBytes(shard uint8, delta int64) {
    v := m.arenaMirror[shard].Add(delta)
    m.arena.WithLabelValues(strconv.Itoa(int(shard))).Set(float64(v))
}
func (m *promMetrics) setArenaBytes(shard uint8, value int64) {
    m.arenaMirror[shard].Store(value)
    m.arena.WithLabelValues(strconv.Itoa(int(shard))).Set(float64(value))
}

/*
   ---------------- Factory ----------------
*/

// newMetricsSink decides which implementation to use.  Caller guarantees that
// shardCount is >0.
func newMetricsSink(shardCount int, reg *prometheus.Registry) metricsSink {
    if reg == nil {
        return noopMetrics{}
    }
    return newPromMetrics(shardCount, reg)
}
