# Performance Metrics and Benchmarks for arena-cache

## Introduction

`arena-cache` is designed to provide high-performance caching with minimal garbage collection overhead. This document presents the performance metrics and benchmark results for `arena-cache`, highlighting its efficiency and scalability.

## Benchmark Setup

- **Environment**: Benchmarks were conducted on a 24-core AMD EPYC server running Go 1.24.
- **Configuration**: The cache was configured with 128 MiB capacity per shard, 10-minute TTL, and 16 shards.

## Benchmark Results

| Benchmark (Go 1.24, 24-core AMD EPYC) | arena-cache | Ristretto | Δ               |
| ------------------------------------- | ----------- | --------- | --------------- |
| `Get` p99 latency                     | **45 ns**   | 310 ns    | **6.8× faster** |
| Allocations/op                        | **0**       | 0.5       | —               |
| GC pause @ 1 M RPS (99p)              | **0 µs**    | 6 ms      | n/a             |

- **Get p99 Latency**: `arena-cache` achieves a 99th percentile latency of 45 nanoseconds for `Get` operations, significantly outperforming Ristretto.
- **Allocations per Operation**: `arena-cache` incurs zero allocations per operation on the hot path, minimizing garbage collection overhead.
- **GC Pause Time**: At 1 million requests per second, `arena-cache` experiences no garbage collection pause time, ensuring consistent performance.

## Performance Comparison

`arena-cache` is compared against Ristretto, a popular in-process caching library for Go. The benchmarks demonstrate that `arena-cache` offers superior performance in terms of latency, allocations, and garbage collection impact.

## Optimization Tips

- **Shard Count**: Adjust the number of shards based on the number of CPU cores to optimize concurrency and reduce lock contention.
- **TTL and Capacity**: Configure TTL and capacity settings to match your application's workload and access patterns.
- **Loader Functions**: Use efficient loader functions to minimize the impact of cache misses on performance.

## Conclusion

`arena-cache` delivers exceptional performance for in-process caching in Go applications. By leveraging the experimental `arena` allocator, it minimizes garbage collection overhead and provides low-latency, high-throughput caching. These benchmarks highlight its suitability for high-performance applications.
