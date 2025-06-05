# API Documentation for arena-cache

## Overview

The `arena-cache` library provides a high-performance, in-process caching solution for Go applications, leveraging the experimental `arena` allocator introduced in Go 1.24. This document outlines the public API available to developers using `arena-cache`.

## Public API

### Cache Creation

- **`New[K comparable, V any](capBytes int64, ttl time.Duration, shards uint8, opts ...Option[K, V]) (*Cache[K, V], error)`**
  - Creates a new cache instance with the specified capacity, time-to-live (TTL), and number of shards.
  - **Parameters**:
    - `capBytes`: Total capacity in bytes for the cache.
    - `ttl`: Duration for which items remain in the cache before expiration.
    - `shards`: Number of shards to divide the cache into, must be a power of two.
    - `opts`: Optional configuration parameters.
  - **Returns**: A pointer to a `Cache` instance and an error if creation fails.

### Cache Operations

- **`Put(ctx context.Context, key K, value V, weight int)`**

  - Inserts a value into the cache with the specified key and weight.
  - **Parameters**:
    - `ctx`: Context for managing request lifetime.
    - `key`: Key associated with the value.
    - `value`: Value to be cached.
    - `weight`: Relative cost or size of the value.

- **`GetOrLoad(ctx context.Context, key K, loader LoaderFunc[K, V]) (V, error)`**
  - Retrieves a value from the cache or loads it using the provided loader function if not present.
  - **Parameters**:
    - `ctx`: Context for managing request lifetime.
    - `key`: Key associated with the value.
    - `loader`: Function to load the value if it is not present in the cache.
  - **Returns**: The cached or loaded value and an error if loading fails.

### Cache Management

- **`Len() int`**

  - Returns the total number of items currently in the cache.

- **`SizeBytes() int64`**

  - Returns the total size in bytes of all items in the cache.

- **`Close()`**
  - Releases resources used by the cache, including freeing all arenas.

## Examples

```go
package main

import (
    "context"
    "fmt"
    "time"

    cache "github.com/Voskan/arena-cache/pkg"
)

type user struct{ ID, Name string }

func main() {
    c, _ := cache.New[string, user](128<<20, 10*time.Minute, 16)
    c.Put(context.Background(), "u123", user{"u123", "Ada"}, 1)
    u, _ := c.GetOrLoad(context.Background(), "u999", func(ctx context.Context, k string) (user, error) {
        return user{ID: k, Name: "generated"}, nil
    })
    fmt.Println(u)
}
```

This example demonstrates creating a cache, inserting a value, and retrieving a value using a loader function.
