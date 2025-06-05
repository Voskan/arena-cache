# Contributing to **arena‑cache**

First off, **thank you** for taking the time to contribute! 🎉 We welcome
issues, questions, improvements, and new features. This guide covers the
workflow, coding conventions, and review process so your contribution can be
merged smoothly.

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Filing Issues](#filing-issues)
3. [Development Workflow](#development-workflow)
4. [Coding Guidelines](#coding-guidelines)
5. [Commit Messages](#commit-messages)
6. [Pull Requests](#pull-requests)
7. [Benchmarks](#benchmarks)
8. [Documentation](#documentation)
9. [Security Reporting](#security-reporting)
10. [Community Conduct](#community-conduct)

---

## Getting Started

```bash
# Fork the repo and clone your fork
$ git clone https://github.com/<you>/arena-cache.git && cd arena-cache

# Install dev tools
$ make install-tools

# Run the full test suite (race + coverage)
$ make test
```

> **Tip:** Use Go 1.24 or later. Toolchain version is enforced by CI.

---

## Filing Issues

- **Bug report?** Include reproduction steps, Go version, hardware/OS, and a
  minimal code sample _or_ failing test.
- **Feature request?** Explain the use‑case and why existing APIs are
  insufficient.
- **Performance regression?** Provide benchmark numbers (`benchstat` output)
  plus CPU / memory profiles if possible.

Before opening a new issue, please search [existing ones](https://github.com/Voskan/arena-cache/issues)
— it might already be tracked.

---

## Development Workflow

1. **Sync main**

   ```bash
   git checkout main && git pull upstream main
   ```

2. **Create feature branch** (`feat/xyz`, `fix/abc` …).
3. **Code!** Ensure:

   - `make lint` passes (`go vet`, `staticcheck`, `golangci-lint`).
   - `make test` passes on your machine.
   - `make bench` shows no significant slow‑down (±5 %).

4. **Commit** (see [Commit Messages](#commit-messages)).
5. **Push & PR** against `main`.
6. GitHub Actions will run CI, benchmarks, and CodeQL. Address any comments.

---

## Coding Guidelines

- **Formatting** — always run `go fmt ./...` (or `goimports`).
- **Imports** — stdlib first, blank line, third‑party, blank, internal.
- **Error handling** — wrap with `%w`; log via `zap.Logger`. Do **not** use
  `fmt.Printf` in library code.
- **Generics** — prefer simple type parameters; avoid unnecessary reflection.
- **Concurrency** — follow the shard model; avoid global locks.
- **Tests**

  - Unit tests next to code (`foo_test.go`).
  - Race detector must pass.
  - Fast (<100 ms each). Heavy benchmarks belong in `bench/`.

---

## Commit Messages

We follow the **Conventional Commits** spec.

```
<type>(<scope>): <subject>

<body>

<footer>
```

- **type** — `feat`, `fix`, `perf`, `docs`, `build`, `ci`, `refactor`, `test`, `chore`.
- **scope** — file or package (`loader`, `shard`, `docs`, `bench`, …).
- **subject** — imperative, no period.
- **body** — _optional_; explain **what** and **why**, not **how**.
- **footer** — `BREAKING CHANGE:` or `Closes #123`.

Example:

```text
perf(shard): reduce hash allocations by unsafe.Slice

Before: 210 ns/op (allocs = 2)
After:   95 ns/op (allocs = 0)
Closes #42
```

---

## Pull Requests

| Checklist                                | Done |
| ---------------------------------------- | ---- |
| `go vet ./...` passes                    | ☐    |
| `make lint` passes                       | ☐    |
| `make test` passes                       | ☐    |
| Benchmarks unaffected (≤ 5 % regression) | ☐    |
| Docs updated (if public API changed)     | ☐    |
| Linked issue / description in PR body    | ☐    |

At least one maintainer must approve; two for large changes.

---

## Benchmarks

Run `make bench` to generate `bench.txt` (3‑run average). For performance
work, include before/after `benchstat` diff in the PR description.

PR CI compares benches against `main` and comments delta.

---

## Documentation

User‑facing docs live in `docs/` (MkDocs). For API changes, update the
relevant `.md` pages and code comments (`godoc`).

Local preview:

```bash
make docs-serve  # http://127.0.0.1:8000
```

---

## Security Reporting

Please email **[security@arena-cache.dev](mailto:security@arena-cache.dev)** for any **vulnerability** or
potential exploit. We operate under [disclosure policy](SECURITY.md).

- Do **not** create public GitHub issues for security problems.
- You will receive acknowledgment within 48 hours and a resolution ETA.

---

## Community Conduct

All interactions are governed by our [Code of Conduct](CODE_OF_CONDUCT.md).
Please be kind and inclusive.

Happy caching! 🚀
