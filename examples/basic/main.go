package main

// examples/basic/main.go spins up a minimal HTTP service that demonstrates how
// to embed arena-cache in a real application.  The service exposes:
//   • PUT /put?key=<k>&val=<v>    — insert a value
//   • GET /get?key=<k>            — fetch or load (on miss)
//   • GET /debug/arena-cache/snapshot — JSON with Len & SizeBytes
//   • GET /metrics                — Prometheus metrics (if built with -tags prom)
//
// Run:
//   go run ./examples/basic
// Then in another terminal:
//   curl "localhost:6060/put?key=foo&val=bar"
//   curl "localhost:6060/get?key=foo"
//   curl "localhost:6060/get?key=baz"        # triggers loader
//   curl "localhost:6060/debug/arena-cache/snapshot"
//
// © 2025 arena-cache authors. MIT License.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	cache "github.com/Voskan/arena-cache/pkg"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// simple value stored in cache

type myVal struct{ Data string }

func main() {
    // Prometheus registry
    reg := prometheus.NewRegistry()

    // Create cache with 64 MiB cap, 5‑min TTL, 8 shards.
    c, err := cache.New[string, myVal](64<<20, 5*time.Minute, 8,
        cache.WithMetrics[string, myVal](reg))
    if err != nil {
        log.Fatalf("cache init: %v", err)
    }

    // Loader: fabricate value on miss.
    loader := func(ctx context.Context, key string) (myVal, error) {
        return myVal{Data: "loaded:" + key}, nil
    }

    mux := http.NewServeMux()

    mux.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
        k := r.URL.Query().Get("key")
        v := r.URL.Query().Get("val")
        if k == "" {
            http.Error(w, "missing key", 400)
            return
        }
        c.Put(r.Context(), k, myVal{Data: v}, 1)
        fmt.Fprintf(w, "OK\n")
    })

    mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
        k := r.URL.Query().Get("key")
        if k == "" {
            http.Error(w, "missing key", 400)
            return
        }
        v, err := c.GetOrLoad(r.Context(), k, loader)
        if err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        fmt.Fprintf(w, "%s\n", v.Data)
    })

    // Snapshot endpoint consumed by arena-cache-inspect.
    mux.HandleFunc("/debug/arena-cache/snapshot", func(w http.ResponseWriter, r *http.Request) {
        snap := map[string]any{
            "items":      c.Len(),
            "arena_bytes": c.SizeBytes(),
        }
        _ = json.NewEncoder(w).Encode(snap)
    })

    // Prometheus metrics.
    mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

    srv := &http.Server{
        Addr:    ":6060",
        Handler: mux,
    }
    log.Println("Listening on http://localhost:6060 …")
    log.Fatal(srv.ListenAndServe())
}
