# Security Policy

Thank you for helping to keep **arena-cache** and its users safe! This document
explains how to report vulnerabilities and what to expect from the project
maintainers.

---

## Supported Versions

Only the latest **minor** release line receives security updates. At any given
time this means the most recent `vX.Y.Z` tag (e.g. `v0.1.*`). Older versions
should be upgraded as soon as possible.

| Version  | Supported | Notes                                   |
| -------- | --------- | --------------------------------------- |
| `main`   | ✅        | development branch, patched immediately |
| `v0.1.x` | ✅        | latest stable                           |
| `< v0.1` | ❌        | no updates                              |

---

## Reporting a Vulnerability

1. **Do _not_ create public issues or pull requests.** This includes "safe"
   repro steps—keep them private.
2. Email **[security@arena-cache.dev](mailto:security@arena-cache.dev)** with the following:

   - A descriptive title (e.g. _Heap-overflow in shard rotation_).
   - Affected version(s) and commit hash if possible.
   - Reproduction steps or proof-of-concept code.
   - Impact assessment (data corruption, RCE, DoS, etc.).
   - Optional: proposed patch or mitigation.

3. Optionally encrypt your report using our PGP key:

   ```
   Fingerprint: 4B2A C1A2 8E76 9D9E C603  8B94 A2AE 4C8C 8731 D2EF
   ```

We acknowledge all vulnerability reports within **48 hours**.

---

## Response Process

| Timeframe | Action                                                                               |
| --------- | ------------------------------------------------------------------------------------ |
| ≤ 48 h    | Acknowledge receipt, request clarification if needed.                                |
| ≤ 7 d     | Confirm vulnerability, evaluate severity (CVSS).                                     |
| ≤ 14 d    | Prepare and test a fix; draft advisory (GHSA).                                       |
| ≤ 21 d    | Coordinate disclosure date with reporter & downstream projects.                      |
| Release   | Tag patched version, publish advisory, credit reporter (unless anonymity requested). |

For critical issues we may accelerate the timeline.

---

## Scope

The policy applies to:

- The Go packages under `github.com/Voskan/arena-cache/...`.
- The CLI tool `arena-cache-inspect`.
- Docker images published under `ghcr.io/<org>/arena-cache-*`.

Issues in **third-party dependencies** should be reported upstream first unless
they only manifest through arena-cache's usage.

---

## Hall of Fame / Credits

We gratefully acknowledge researchers and users who responsibly disclose
security issues. See `Hall of Fame` section in future advisories.

---

## Disclaimer

Security is a process, not a guarantee. We make every effort to fix issues
promptly, but the software is provided **"as is"** without warranty of any
kind. See the [LICENSE](LICENSE) file for details.
