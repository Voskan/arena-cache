package main

// examples/disk_eject/main.go demonstrates how to combine arena-cache with a
// second-level on-disk store (BadgerDB).  Evicted items are written to Badger
// via EjectCallback, and the loader first consults Badger before generating a
// fallback value.
//
// Endpoints:
//   PUT /put?key=<k>&val=<v>   – insert into L1 cache.
//   GET /get?key=<k>           – retrieve: L1 → Badger → generated.
//   GET /stats                 – JSON: items, arena_bytes, badger_keys.
//
// Run:
//   go run ./examples/disk_eject
// ---------------------------------------------------------------
// © 2025 arena-cache authors. MIT License.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	cache "github.com/Voskan/arena-cache/pkg"
	badger "github.com/dgraph-io/badger/v4"
)

func main() {
    // Open Badger (embedded key-value DB) in ./l2 directory.
    bdb, err := badger.Open(badger.DefaultOptions("./l2").WithLogger(nil))
    if err != nil {
        log.Fatalf("badger: %v", err)
    }
    defer bdb.Close()

    // EjectCallback writes evicted entries to Badger.
    eject := func(key string, val string, _ cache.EjectReason) {
        if err := bdb.Update(func(txn *badger.Txn) error {
            return txn.Set([]byte(key), []byte(val))
        }); err != nil {
            log.Printf("badger set err: %v", err)
        }
    }

    // Create L1 cache.
    c, err := cache.New[string, string](32<<20 /*32MiB*/, 2*time.Minute, 4,
        cache.WithEjectCallback[string, string](eject))
    if err != nil {
        log.Fatalf("cache init: %v", err)
    }
    defer c.Close()

    // Loader: try Badger, else fabricate.
    loader := func(ctx context.Context, key string) (string, error) {
        var v string
        err := bdb.View(func(txn *badger.Txn) error {
            item, err := txn.Get([]byte(key))
            if err != nil {
                return err
            }
            return item.Value(func(b []byte) error {
                v = string(b)
                return nil
            })
        })
        if err == nil {
            return v, nil
        }
        // Not in Badger: generate value and persist.
        v = "gen:" + key
        _ = bdb.Update(func(txn *badger.Txn) error {
            return txn.Set([]byte(key), []byte(v))
        })
        return v, nil
    }

    mux := http.NewServeMux()

    mux.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
        k := r.URL.Query().Get("key")
        v := r.URL.Query().Get("val")
        if k == "" {
            http.Error(w, "missing key", 400)
            return
        }
        c.Put(r.Context(), k, v, 1)
        fmt.Fprintln(w, "OK")
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
        fmt.Fprintln(w, v)
    })

    mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
        var badgerKeys uint64
        _ = bdb.View(func(txn *badger.Txn) error {
            it := txn.NewIterator(badger.DefaultIteratorOptions)
            for it.Rewind(); it.Valid(); it.Next() {
                badgerKeys++
            }
            it.Close()
            return nil
        })
        snap := map[string]any{
            "l1_items":     c.Len(),
            "l1_arena_bytes": c.SizeBytes(),
            "l2_keys":      badgerKeys,
        }
        _ = json.NewEncoder(w).Encode(snap)
    })

    srv := &http.Server{Addr: ":7070", Handler: mux}
    log.Println("Listening http://localhost:7070 …")
    if err := srv.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
