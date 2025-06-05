# Changelog

All notable changes to **arena-cache** will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- Adaptive rotation algorithm prototype behind build tag `adaptive_rotate`.
- `WithPartition()` functional option (experimental).

### Changed

- Prometheus histogram buckets tuned for high‑cardinality shards.

### Fixed

- Race condition in inspector CLI when context cancels before HTTP connect.

---

## [v0.1.0] – 2025‑06‑05

### Added

- **Initial public release** 🎉

  - Zero‑GC arena generations with TTL.
  - CLOCK‑Pro capacity eviction.
  - Sharded map with per‑shard metrics.
  - Inspector CLI (`arena-cache-inspect`).
  - Prometheus metrics and OpenTelemetry spans.
  - Examples: `basic`, `disk_eject`.
  - GitHub Actions CI, Release, Docs, CodeQL.
  - Docker & Compose playground.
  - Benchmarks suite.

### Security

- CodeQL static analysis enabled.

---

[unreleased]: https://github.com/Voskan/arena-cache/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/Voskan/arena-cache/releases/tag/v0.1.0
