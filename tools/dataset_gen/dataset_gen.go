// Move this file to tools/dataset_gen to separate it from the bench package.

package main

// dataset_gen.go is a tiny helper utility to generate deterministic key
// datasets for standalone benchmarking of arena-cache (outside `go test`).
// It emits newline-separated uint64 numbers which can later be passed to
// service load-testers or external benchmarking suites.
//
// Usage:
//   go run bench/dataset_gen.go -n 1000000 -dist=zipf -seed=42 -out keys.txt
//
// Flags:
//   -n       number of keys to generate (default 1e6)
//   -dist    distribution: "uniform" or "zipf" (default uniform)
//   -zipfs   Zipf s parameter (>1)  (default 1.2)
//   -zipfv   Zipf v parameter (>1)  (default 1.0)
//   -seed    RNG seed (default current time)
//   -out     output file (default stdout)
//
// The program is *embarassingly simple* but placed under version control so
// that any contributor can regenerate the exact dataset used in performance
// regressions hunting.
//
// © 2025 arena-cache authors. MIT License.

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
)

func main() {
    var (
        n       = flag.Int("n", 1_000_000, "number of keys to generate")
        dist    = flag.String("dist", "uniform", "distribution: uniform or zipf")
        zipfS   = flag.Float64("zipfs", 1.2, "zipf s parameter (>1)")
        zipfV   = flag.Float64("zipfv", 1.0, "zipf v parameter (>1)")
        seedVal = flag.Int64("seed", time.Now().UnixNano(), "PRNG seed")
        outPath = flag.String("out", "", "output file (default stdout)")
    )
    flag.Parse()

    rnd := rand.New(rand.NewSource(*seedVal))

    var gen func() uint64
    switch *dist {
    case "uniform":
        gen = rnd.Uint64
    case "zipf":
        if *zipfS <= 1.0 || *zipfV <= 0 {
            fmt.Fprintln(os.Stderr, "zipfs must be >1 and zipfv >0")
            os.Exit(1)
        }
        z := rand.NewZipf(rnd, *zipfS, *zipfV, ^uint64(0))
        gen = z.Uint64
    default:
        fmt.Fprintln(os.Stderr, "unknown dist:", *dist)
        os.Exit(1)
    }

    var out *os.File
    var err error
    if *outPath == "" {
        out = os.Stdout
    } else {
        out, err = os.Create(*outPath)
        if err != nil {
            fmt.Fprintln(os.Stderr, "cannot create file:", err)
            os.Exit(1)
        }
        defer out.Close()
    }

    w := bufio.NewWriterSize(out, 1<<20)
    defer w.Flush()

    for i := 0; i < *n; i++ {
        fmt.Fprintln(w, gen())
    }
}
