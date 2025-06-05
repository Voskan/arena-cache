# Design Decisions in arena-cache

## Introduction

The `arena-cache` project is designed to provide a high-performance, in-process caching solution for Go applications. This document outlines the key design decisions made during the development of `arena-cache`, along with the rationale behind these choices.

## Key Design Decisions

### Use of Arena Allocator

- **Decision**: Leverage the experimental `arena` allocator introduced in Go 1.24 to manage memory outside the garbage-collected heap.
- **Rationale**: Using arenas allows for efficient memory management with minimal garbage collection overhead, making `arena-cache` suitable for high-throughput applications.

### Sharded Architecture

- **Decision**: Divide the cache into multiple shards, each operating independently.
- **Rationale**: Sharding reduces lock contention and improves concurrency, allowing `arena-cache` to scale effectively across multiple CPU cores.

### CLOCK-Pro Replacement Algorithm

- **Decision**: Implement the CLOCK-Pro algorithm for cache replacement.
- **Rationale**: CLOCK-Pro provides a balance between recency and frequency of access, making it well-suited for a wide range of workloads. It offers predictable performance and efficient cache management.

### Generational Ring

- **Decision**: Use a generational ring to manage the lifecycle of arenas.
- **Rationale**: The generational ring allows for efficient TTL and capacity-based eviction, ensuring that stale or excess data is removed with minimal overhead.

### Minimal API Surface

- **Decision**: Expose a minimal API surface to users, focusing on core caching functionality.
- **Rationale**: A minimal API reduces complexity and makes it easier for developers to integrate `arena-cache` into their applications. It also simplifies maintenance and future enhancements.

## Trade-offs and Alternatives

### Alternative Cache Replacement Algorithms

- **Considered**: LRU, LFU, and other cache replacement algorithms.
- **Reason for Rejection**: While these algorithms have their merits, CLOCK-Pro was chosen for its ability to balance recency and frequency, providing better performance for the target use cases.

### In-Process vs. Distributed Caching

- **Considered**: Implementing a distributed caching solution.
- **Reason for Rejection**: `arena-cache` is designed to be an in-process cache, focusing on high performance and low latency. Distributed caching introduces additional complexity and network overhead, which are not aligned with the project's goals.

## Future Directions

- Explore adaptive rotation strategies based on access patterns.
- Investigate tiered arenas for different priority classes.
- Consider native support for OpenTelemetry metrics.

## Conclusion

The design decisions made in `arena-cache` are aimed at providing a scalable, efficient, and easy-to-use caching solution for Go applications. By leveraging the latest advancements in Go's memory management, `arena-cache` delivers high throughput and low latency, making it suitable for a wide range of use cases.
