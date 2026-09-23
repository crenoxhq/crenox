# Contributing to Crenox

Thank you for your interest in contributing to Crenox. This document outlines the process for reporting issues, proposing enhancements, and submitting code contributions.

Crenox is an open-source security tool maintained by **CrenoxHQ**. We welcome contributions from the community while upholding strict standards of performance, memory efficiency, and code quality.

---

## 1. Code of Conduct and Expectations

Contributors are expected to treat all community members with professional respect. Technical critiques should remain constructive and focused on code, algorithms, and security outcomes.

---

## 2. Reporting Issues

### 2.1 False Positives and False Negatives
Because Crenox operates on a three-tier detection pipeline (Aho-Corasick trie, Shannon entropy, and context heuristics), reports regarding incorrect detections are treated with high priority:
* **False Positive:** A safe, non-sensitive token or public key incorrectly flagged as a secret.
* **False Negative:** A real secret or credential missed by the scanner.

When reporting detection issues:
1. Do not post real, active production secrets. Always redact or substitute sensitive characters with dummy placeholders while preserving token length, character set, and format prefix.
2. Provide the exact file path, file extension, and a minimal reproducible snippet.
3. State the command used (for example: `crenox run` vs. `crenox scan .`).

### 2.2 Security Vulnerabilities
If you discover a security vulnerability within Crenox itself, please report it privately through GitHub Security Advisories:
https://github.com/crenoxhq/crenox/security/advisories/new

---

## 3. Submitting Contributions

### 3.1 Architecture and Engineering Standards
Crenox prioritizes execution speed and predictable memory utilization. All code submitted to the core scanning path must adhere to the following principles:
* **Zero Allocations on Hot Paths:** Operations inside `internal/trie`, `internal/entropy`, and per-token scanning loops must avoid heap allocations (`0 B/op, 0 allocs/op`).
* **Resource Re-use:** Utilize `sync.Pool` for buffer allocations where streaming or intermediate data processing is required.
* **Bounded Concurrency:** Concurrency must be bounded (e.g., matching CPU core counts) and must not spawn uncontrolled goroutines per file or token.
* **Safe Regular Expressions:** Custom pattern rules and internal automata must use `regexp.Compile` with proper error handling, never `regexp.MustCompile` in runtime paths.

### 3.2 Testing and Verification
Every functional change, new signature, or bug fix must include corresponding tests:
* Run the complete test suite:
  ```bash
  go test -v ./...
  ```
* Verify benchmark performance and allocations:
  ```bash
  go test -bench=. ./...
  ```
* Ensure all tests pass with the race detector:
  ```bash
  go test -race ./...
  ```

### 3.3 Blocking Policy and Exit Invariants
Crenox enforces a deterministic, fail-closed security model:
* **Zero-Tolerance Blocking:** Crenox uses a zero-tolerance blocking policy. Any detected finding, regardless of severity (`CRITICAL`, `HIGH`, `MEDIUM`, or `LOW`), blocks the commit with exit code 1.
* **Fail-Closed Scan Integrity:** Any unreadable file, failed Git operation, permission denial, or incomplete scan also blocks the commit with exit code 1. A clean result (`exit 0`) is valid only when 100% of eligible files have been scanned successfully with zero findings.
* **Status Separation:**
  - `blocked` (`exit 1`): One or more secrets were detected.
  - `scan_error` (`exit 1`): One or more files could not be read or Git operations failed.
  - `clean` (`exit 0`): All files were scanned without errors and zero secrets were detected.
* **Explicit Bypass Only:** False positive suppression is allowed strictly via explicit inline comments (`crenox:ignore` on the preceding line) or explicit path/string entries in `.crenox.yaml`.

---

## 4. Legal Framework and Dual-Licensing

Crenox is distributed to the open-source community under the **GNU Affero General Public License v3.0 (AGPLv3)**.

To sustain ongoing research, development, and engineering infrastructure, CrenoxHQ also provides commercial and enterprise licensing to organizations requiring non-copyleft distribution, proprietary embedding, or enterprise SLAs.

All contributors must review and agree to the **[Contributor License Agreement (CLA)](CLA.md)** before their pull requests can be merged. The CLA grants CrenoxHQ the legal standing to license and distribute the software across both open-source and commercial channels, ensuring the project's long-term sustainability.

---

## 5. Pull Request Workflow

### 5.1 Branch Policy
* All development and pull requests must target the `dev` branch.
* The `main` branch is reserved strictly for tagged production releases and audited artifacts. Pull requests targeting `main` directly will be redirected to `dev`.

### 5.2 Prerequisites
* Go 1.23 or later installed.
* Git 2.30+ installed.

### 5.3 Step-by-Step Submission
1. Fork the repository and create a descriptive feature or bugfix branch from `dev` (e.g., `git checkout -b feat/new-pattern dev`).
2. Implement your changes, following standard Go conventions (`gofmt`, `go vet`).
3. Add unit tests and verify that existing test suites and benchmarks pass with zero heap allocations on hot paths.
4. Ensure your commit messages are clear, descriptive, and follow Conventional Commits (e.g., `fix:`, `feat:`, `perf:`, `docs:`).
5. Open a Pull Request targeting the `dev` branch.
6. Fill out the pull request template completely, confirming CLA acceptance and zero-allocation compliance.
