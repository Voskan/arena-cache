# Architecture Overview of arena-cache

## Introduction

`arena-cache` is designed to provide a high-performance, in-process caching solution for Go applications. It leverages the experimental `arena` allocator introduced in Go 1.24 to minimize garbage collection overhead. This document provides an overview of the architecture and key components of `arena-cache`.

## Key Components

### Shards

- **Description**: The cache is divided into multiple shards to reduce lock contention and improve concurrency. Each shard operates independently and manages its own set of keys and values.
- **Concurrency**: Shards use RWMutex to ensure thread-safe operations while minimizing lock contention.

### Arenas

- **Description**: Arenas are memory regions outside the Go garbage-collected heap. They are used to store cached values, allowing for efficient memory management and minimal GC overhead.
- **Lifecycle**: Arenas are created and freed in bulk, providing O(1) memory release when a generation is rotated out.

### CLOCK-Pro Algorithm

- **Description**: `arena-cache` uses the CLOCK-Pro algorithm for cache replacement. This algorithm provides a balance between recency and frequency of access, making it suitable for a wide range of workloads.
- **Implementation**: Each shard maintains a CLOCK-Pro ring to manage cache entries and determine which entries to evict when necessary.

### Generational Ring

- **Description**: The generational ring is a circular buffer of arenas. It manages the lifecycle of arenas, including creation, rotation, and freeing.
- **TTL and Capacity**: The ring rotates based on time-to-live (TTL) and capacity constraints, ensuring that stale or excess data is efficiently removed.

## Data Flow

1. **Request Handling**: Incoming requests are distributed across shards based on a hash of the key.
2. **Cache Operations**: Each shard handles cache operations independently, using its own arena and CLOCK-Pro ring.
3. **Eviction and Rotation**: When a shard's arena reaches its capacity or TTL, the generational ring rotates, freeing the oldest arena and creating a new one.

## Diagrams

Below is a high-level diagram illustrating the architecture of `arena-cache`:

```
┌─────────┐ requests ┌──────────┐   CLOCK-Pro ┌──────────┐ rotate  ┌──────────┐ free ┌───────┐
│  App    ├─────────▶│  Shard N ├────────────▶│  Shard 0 ├────────▶│  Arena 0 ├──────▶│  GC  │
└─────────┘          └──────────┘   …         └──────────┘         └──────────┘      └───────┘
      ▲                      ▲                                             │
      │                      └── Prom metrics, OTel traces ▲                │
      │                                           snapshot │                │
      └──────────────────────────────────────── arena-cache-inspect ◀────────┘
```

## Conclusion

`arena-cache` is designed to provide a scalable and efficient caching solution for Go applications. Its architecture leverages the latest advancements in Go's memory management to deliver high throughput and low latency, making it suitable for a wide range of use cases.
