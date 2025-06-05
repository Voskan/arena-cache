package main

// main.go implements the arena‑cache inspector CLI: it parses command‑line
// flags, fetches diagnostic data from a target process exposing the
// arena‑cache debug endpoint, and prints it either as pretty text or JSON.  It
// also supports periodic watch mode and pprof snapshot download.
//
// The target Go service is expected to expose:
//   • GET /debug/arena-cache/snapshot  – JSON payload with cache statistics.
//   • GET /debug/pprof/{heap,goroutine} – standard pprof handlers (net/http/pprof).
//
// The snapshot object is intentionally generic; we decode into map[string]any
// to avoid version skew between CLI and library.
//
// Build-time flag: `-ldflags "-X main.version=vX.Y.Z"` is set by GoReleaser.
// ---------------------------------------------------------------
// © 2025 arena-cache authors. MIT License.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "dev"

func main() {
    opts := parseFlags()

    if opts.version {
        fmt.Println(version)
        return
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle SIGINT/SIGTERM for graceful exit.
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sig
        cancel()
    }()

    // pprof dump takes precedence over watch/json.
    if opts.heapProfile != "" {
        if err := downloadProfile(ctx, opts.target, "heap", opts.heapProfile); err != nil {
            fatal(err)
        }
        return
    }
    if opts.goroutineProfile != "" {
        if err := downloadProfile(ctx, opts.target, "goroutine", opts.goroutineProfile); err != nil {
            fatal(err)
        }
        return
    }

    if opts.watch {
        ticker := time.NewTicker(opts.interval)
        defer ticker.Stop()
        for {
            if err := dumpOnce(ctx, opts); err != nil {
                fmt.Fprintln(os.Stderr, "error:", err)
            }
            select {
            case <-ticker.C:
                continue
            case <-ctx.Done():
                return
            }
        }
    }

    // one‑shot
    if err := dumpOnce(ctx, opts); err != nil {
        fatal(err)
    }
}

/* -------------------------------------------------------------------------
   Helpers
   ------------------------------------------------------------------------- */

func dumpOnce(ctx context.Context, opts *options) error {
    snap, err := fetchSnapshot(ctx, opts.target)
    if err != nil {
        return err
    }

    if opts.json {
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(snap)
    }
    return prettyPrint(snap)
}

func fetchSnapshot(ctx context.Context, base string) (map[string]any, error) {
    url := base + "/debug/arena-cache/snapshot"
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    res, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer res.Body.Close()
    if res.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status %s", res.Status)
    }
    var data map[string]any
    if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
        return nil, err
    }
    return data, nil
}

func prettyPrint(data map[string]any) error {
    // naive pretty printer – assume common top‑level fields
    fmt.Printf("Hits:     %v\n", data["hits_total"])
    fmt.Printf("Misses:   %v\n", data["misses_total"])
    fmt.Printf("Evictions:%v\n", data["evictions_total"])
    fmt.Printf("Arena MB: %.2f\n", toFloat(data["arena_bytes"])/1_048_576)
    return nil
}

func toFloat(v any) float64 {
    switch t := v.(type) {
    case float64:
        return t
    case int64:
        return float64(t)
    case json.Number:
        f, _ := t.Float64()
        return f
    default:
        return 0
    }
}

func downloadProfile(ctx context.Context, base, name, path string) error {
    url := fmt.Sprintf("%s/debug/pprof/%s", base, name)
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    res, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer res.Body.Close()
    if res.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status %s", res.Status)
    }

    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()

    _, err = io.Copy(f, res.Body)
    if err != nil {
        return err
    }
    fmt.Printf("%s profile saved to %s\n", name, path)
    return nil
}

func fatal(err error) {
    fmt.Fprintln(os.Stderr, "arena-cache-inspect:", err)
    os.Exit(1)
}
