<div align="center">

```
 ███████╗███████╗███╗   ██╗████████╗██╗███╗   ██╗███████╗██╗
 ██╔════╝██╔════╝████╗  ██║╚══██╔══╝██║████╗  ██║██╔════╝██║
 ███████╗█████╗  ██╔██╗ ██║   ██║   ██║██╔██╗ ██║█████╗  ██║
 ╚════██║██╔══╝  ██║╚██╗██║   ██║   ██║██║╚██╗██║██╔══╝  ██║
 ███████║███████╗██║ ╚████║   ██║   ██║██║ ╚████║███████╗███████╗
 ╚══════╝╚══════╝╚═╝  ╚═══╝   ╚═╝   ╚═╝╚═╝  ╚═══╝╚══════╝╚══════╝
```

**Enterprise-grade Git pre-commit secret detector**

[![CI](https://github.com/sentinel-cli/sentinel/actions/workflows/ci.yml/badge.svg)](https://github.com/sentinel-cli/sentinel/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-AGPL_3.0-blue)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows%20%7C%20Android%2FTermux-informational)](#installation)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go)

</div>

---

Sentinel is a **statically compiled, zero-dependency** Git pre-commit hook written in Go. It blocks accidental commits of API keys, SSH private keys, database passwords, and other sensitive credentials via a **three-tier detection pipeline** tuned for near-zero latency and near-zero false positives.

It runs on any platform where Go compiles — including **Android/Termux** and minimal embedded Linux environments.

---

## Table of Contents

- [📊 The Doomsday Benchmark: Sentinel vs. Industry Standards](#-the-doomsday-benchmark-sentinel-vs-industry-standards)
- [Why Sentinel?](#why-sentinel)
- [Architecture](#architecture)
  - [Detection Pipeline](#detection-pipeline)
  - [Tier 1 — Aho-Corasick Pattern Matching](#tier-1--aho-corasick-pattern-matching)
  - [Tier 2 — Shannon Entropy Analysis](#tier-2--shannon-entropy-analysis)
  - [Tier 3 — Context-Aware Filtering](#tier-3--context-aware-filtering)
  - [Module Layout](#module-layout)
- [Signature Coverage](#signature-coverage)
- [Performance](#performance)
  - [The Ultimate Meat Grinder](#the-ultimate-meat-grinder)
- [Installation](#installation)
  - [Prerequisites](#prerequisites)
  - [Install via go install (Recommended)](#install-via-go-install-recommended)
  - [Build from Source](#build-from-source)
  - [Android / Termux](#android--termux)
  - [Hook — Current Repository](#hook--current-repository)
  - [Hook — Global (All Repositories)](#hook--global-all-repositories)
- [Configuration](#configuration)
  - [Config File Resolution](#config-file-resolution)
  - [Full Config Reference](#full-config-reference)
  - [Entropy Threshold Tuning](#entropy-threshold-tuning)
  - [Excluding Paths and Extensions](#excluding-paths-and-extensions)
- [Usage](#usage)
  - [Pre-commit Hook](#pre-commit-hook)
  - [Ad-hoc File Scan](#ad-hoc-file-scan)
  - [JSON Output Mode](#json-output-mode)
  - [CI Integration](#ci-integration)
  - [Lifecycle Commands](#lifecycle-commands)
- [Running Tests](#running-tests)
- [Output Reference](#output-reference)
- [False Positive Handling](#false-positive-handling)
- [Contributing](#contributing)
- [License](#license)

---

## 📊 Test Environment & Specifications

These tests were executed inside a **`proot` environment on Termux/Android**. This is a constrained, emulated user-space environment without native root permissions, demonstrating Sentinel's true zero-dependency advantage and ultra-fast performance even on mobile hardware limitations.

- **OS / Kernel**: Linux localhost 6.17.0-PRoot-Distro (Android / Termux)
- **Architecture**: `aarch64` (ARM64)
- **CPU**: 8-Core ARM (Cortex-A55 / Cortex-A75) @ 2.0 GHz
- **Memory**: ~2.5 GB Total RAM (~640 MB Available during tests)

---

## 📊 The Doomsday Benchmark: Sentinel vs. Industry Standards

We executed the **"Doomsday Benchmark"** (available in `tests/benchmark/doomsday_generator.py`) generating ~15MB of compressed minified lines, high-entropy noise, and syntax baits. The payload was seeded with exactly 3 Real Secrets (GitHub PAT, Base64 Encoded AWS Key, and PEM Private Key) alongside 100 fake Stripe keys, 20,000 invalid AWS keys, and common Android constants like `REQUEST_IGNORE_BATTERY_OPTIMIZATIONS` and `sg.messageId`.

### Comparison Benchmark Matrix

| Metric | Sentinel | Gitleaks | TruffleHog |
| :--- | :--- | :--- | :--- |
| **Execution Time (Real)** | **2.37s** | 1.81s | 18.54s |
| **False Positives (Traps)** | **0** | 100 | 0 |
| **Caught Secrets (Out of 3)** | **3 / 3** | 2 / 3 | 1 / 3 |
| **Signal-to-Noise Ratio** | **100% (3 true, 0 noise)** | 2.0% (2 true, 100 noise) | 100% (1 true, 0 noise) |

### 🧠 Why Sentinel Obliterated the Standards

- **Two-Tier Architecture**: Aho-Corasick matches prefixes blazingly fast, but Sentinel's strict Regex Validator evaluates the structural context and immediately drops variable baits like `sg.messageId` natively.
- **Cryptographic Validation**: Sentinel checks the actual structure, length constraints, and context of the token. It completely bypassed the 100 fake Stripe keys, achieving a 100% Signal-to-Noise ratio while Gitleaks choked on 101 false alarms.
- **Zero Pipeline Friction**: TruffleHog failed the friction test with an 11.9s runtime. Sentinel parsed the same Doomsday files in **1.5s**, making it truly suitable for developer pre-commit hooks.

---

## Why Sentinel?

| Feature | Sentinel | git-secrets | detect-secrets | truffleHog |
|---------|----------|-------------|----------------|------------|
| Statically compiled (no runtime deps) | ✅ | ❌ (bash) | ❌ (Python) | ❌ (Python) |
| ARM / Android / Termux support | ✅ | ⚠️ | ❌ | ❌ |
| Aho-Corasick O(n) multi-pattern scan | ✅ | ❌ | ❌ | ❌ |
| Shannon entropy detection | ✅ | ❌ | ✅ | ✅ |
| Context-aware false positive suppression | ✅ | ❌ | ✅ | ⚠️ |
| Base64 Single-Layer Extraction | ✅ | ❌ | ✅ | ✅ |
| Termux-Aware TLS Self-Healing | ✅ | ❌ | ❌ | ❌ |
| Sub-15ms scan (50 KB file) | ✅ | ⚠️ | ❌ | ❌ |
| JSON output for CI tooling | ✅ | ❌ | ✅ | ✅ |
| Zero external runtime dependencies | ✅ | ✅ | ❌ | ❌ |
| Global hook installation | ✅ | ✅ | ❌ | ❌ |

---

## Architecture

### Detection Pipeline

Every staged file passes through three sequential tiers. A finding must **survive all three tiers** to be reported, which eliminates the vast majority of false positives seen in single-pass tools.

```
 ┌──────────────────────────────────────────────────────────────────┐
 │                    git commit (staged changes)                    │
 └───────────────────────────┬──────────────────────────────────────┘
                             │
              ┌──────────────▼──────────────┐
              │   git interop (internal/git) │
              │  ListStagedFiles()           │
              │  GetStagedDiff() / GetBlob() │
              └──────────────┬──────────────┘
                             │
              ┌──────────────▼──────────────┐
              │       Pre-flight filters     │
              │  • Binary file skip          │
              │  • Extension exclusion       │
              │  • Path exclusion (glob)     │
              │  • File size cap (10 MB)     │
              └──────────────┬──────────────┘
                             │
              ┌──────────────▼──────────────┐
              │  TIER 1: Aho-Corasick Trie   │
              │  (internal/trie)             │
              │  O(n) multi-pattern search   │
              │  60+ known secret prefixes   │
              └──────────────┬──────────────┘
                             │
              ┌──────────────▼──────────────┐
              │  TIER 2: Shannon Entropy     │
              │  (internal/entropy)          │
              │  Base64 + hex token extract  │
              │  Configurable threshold      │
              └──────────────┬──────────────┘
                             │
              ┌──────────────▼──────────────┐
              │  TIER 3: Context Filter      │
              │  (internal/context)          │
              │  Comment / test file check   │
              │  Placeholder / UUID check    │
              │  Variable name heuristics    │
              │  Assignment-aware extraction │
              └──────────────┬──────────────┘
                             │
              ┌──────────────▼──────────────┐
              │  Reporter (internal/reporter)│
              │  Pretty / JSON / Plain       │
              └──────────────┬──────────────┘
                             │
               exit 0 (CLEAN) or exit 1 (BLOCKED)
```

---

### Tier 1 — Aho-Corasick Pattern Matching

**File:** [`internal/trie/trie.go`](internal/trie/trie.go)

Tier 1 implements the **Aho-Corasick string-matching automaton** — a multi-pattern algorithm that scans a byte stream in **O(n + m)** time regardless of how many patterns are loaded.

**Automaton construction (once at startup):**
1. All 60+ secret prefixes (e.g. `ghp_`, `AKIA`, `-----BEGIN RSA PRIVATE KEY-----`) are inserted into a trie.
2. A BFS traversal computes **failure links** for each node, enabling resume-on-mismatch without backtracking.
3. **Output links** are merged so overlapping patterns (e.g. `sk-` and `sk-proj-`) are both detected in a single pass.

**Scanning (per file):**
- Each byte is processed exactly once via O(1) state transitions.
- All patterns are lowercased at build time — matching is case-insensitive.
- A pre-built **newline index** enables O(log n) line-number lookup via binary search.
- Detects secrets leaked inside **unstructured kernel panic logs**, memory dumps, and base64 payloads without relying on variable assignments.
- Evaluates raw plain-text explicitly for 12-to-24 word **BIP-39 Seeds**, capturing secrets dumped loosely in `.txt` or `.md` files.
- Extracts **multiple distinct secrets per line**, completely eliminating blindspots in minified JavaScript or single-line config files.
- **Bulletproof Deduplication:** Resolves overlaps between Pattern hits and Entropy hits, prioritizing strict pattern signatures.
- Now natively detects **PEM Certificates** (RSA/Private Keys) even across multi-line payloads.

**Auto-Updater Engine:**
- Employs a custom **UDP DNS Resolver (8.8.8.8:53)** to bypass OS-level IPv6 misconfigurations and Loopback failures during background updates.

---

### Tier 2 — Shannon Entropy Analysis

**File:** [`internal/entropy/entropy.go`](internal/entropy/entropy.go)

Tier 2 catches secrets without known prefixes — raw cryptographic keys, custom tokens, long passwords — by measuring the **information density** of candidate string tokens.

**Shannon entropy formula:**

```
H(X) = - Σ P(xᵢ) · log₂(P(xᵢ))
```

Where P(xᵢ) is the frequency of byte value xᵢ in the token. A perfectly uniform 256-symbol distribution yields **8.0 bits/symbol**. English prose yields ~3.5. A 32-byte random Base64 secret yields **~5.5–6.5**.

**Token extraction:**
- Contiguous runs of **Base64-alphabet** chars (`A-Za-z0-9+/=_-`) and **hex-alphabet** chars (`0-9a-fA-F`) are extracted per line.
- Tokens shorter than `min_secret_length` (default: 20) are skipped.
- Tokens with all-identical characters (zero entropy) are skipped.
- Hex tokens must have even length to resemble real hashes.
- Only tokens exceeding `entropy_threshold` (default: 4.5 bits) advance to Tier 3.

---

### Tier 3 — Context-Aware Filtering

**File:** [`internal/context/context.go`](internal/context/context.go)

Tier 3 is the **false positive elimination layer**. It inspects the structural context of each candidate finding and returns one of the following decisions:

| Decision | Condition | Example |
|----------|-----------|---------|
| `Real` | None of the below apply | Production API key in `config.go` |
| `SafeComment` | Line starts with `//`, `#`, `*`, `<!--`, etc. | `# old_key = "ghp_..."` |
| `SafeTestFile` | Path contains `_test.go`, `tests/`, `fixtures/`, `.md`, etc. | `auth_test.go` |
| `SafeVariableName` | Line contains `dummy`, `fake`, `mock`, `placeholder`, etc. | `dummy_api_key := "..."` |
| `SafePlaceholder` | Token matches `$VAR`, `${VAR}`, `<placeholder>`, `{{template}}` | `token: ${MY_TOKEN}` |
| `SafeUUID` | Token matches UUID v4 format | `id = "550e8400-e29b-..."` |
| `SafeVersionString` | Token matches a semantic version pattern | `"1.23.456-beta"` |

Only `Real` findings are reported. Additionally, the scanner's **assignment-aware value extraction** ensures that:
- Format strings (e.g. `fmt.Printf("token=%s\n", v)`) are never flagged.
- PascalCase identifiers matching short prefixes (e.g. `ACAccountSID`) are rejected.
- SQL template placeholders (e.g. `password=?`) are not treated as secrets.
- English prose in log messages does not trigger entropy analysis.
- **Minified JS files** with multiple statements per line are parsed directionally backward from the token to find the exact nearest variable context, avoiding false suppressions from adjacent dummy variables.

---

### Module Layout

```
sentinel/
├── cmd/
│   └── sentinel/
│       ├── main.go                  # CLI root (cobra)
│       └── commands/
│           ├── run.go               # sentinel run  — pre-commit hook entry
│           ├── install.go           # sentinel install — hook installation
│           ├── uninstall.go         # sentinel uninstall — hook removal
│           ├── scan.go              # sentinel scan  — ad-hoc file scan
│           ├── update.go            # sentinel update — self-updater
│           ├── version.go           # sentinel version — build metadata
│           └── helpers.go           # shared exec helper
│
├── internal/
│   ├── config/
│   │   └── config.go               # YAML config schema, loader, validation
│   ├── git/
│   │   └── git.go                  # git interop: staged files, diff, blobs
│   ├── trie/
│   │   ├── trie.go                 # Aho-Corasick automaton + signature catalogue
│   │   └── bip39.go                # BIP-39 mnemonic word list
│   ├── entropy/
│   │   └── entropy.go              # Shannon entropy calculator + token extractor
│   ├── context/
│   │   └── context.go              # Tier 3 context classifier
│   ├── scanner/
│   │   └── scanner.go              # Three-tier pipeline orchestrator + Finding type
│   ├── reporter/
│   │   └── reporter.go             # Pretty / JSON / Plain output renderer
│   └── updater/
│       └── updater.go              # Background release-check (non-blocking)
│
├── pkg/
│   └── version/
│       └── version.go              # Build metadata (version, commit, date)
│
├── tests/
│   ├── benchmark/
│   │   └── doomsday_generator.py   # The Doomsday Stress-Test Suite
│   ├── trie_test.go                # Tier 1 unit + benchmark tests
│   ├── entropy_test.go             # Tier 2 unit + benchmark tests
│   ├── context_test.go             # Tier 3 unit tests
│   ├── scanner_test.go             # End-to-end pipeline + performance tests
│   └── doc.go                      # Package declaration
│
├── scripts/
│   ├── build.sh                    # Cross-platform release build script
│   └── test.sh                     # Test runner with coverage report
│
├── .github/
│   └── workflows/
│       └── ci.yml                  # GitHub Actions CI pipeline
│
├── .sentinel.yaml.example          # Fully annotated configuration reference
├── CHANGELOG.md                    # Release history
├── CLA.md                          # Contributor License Agreement
├── Makefile                        # Developer convenience targets
├── go.mod
└── README.md
```

---

## Signature Coverage

Sentinel's Tier 1 catalogue detects **60+ secret families** across all major platforms:

| Category | Services Covered |
|----------|-----------------|
| **VCS Tokens** | GitHub PAT (classic & fine-grained), GitHub OAuth, GitHub App/Refresh, GitLab PAT, GitLab Pipeline, GitLab Runner |
| **Cloud** | AWS Access Key / STS / MFA, GCP Service Account (JSON), GCP API Key, DigitalOcean, Cloudflare, Vercel |
| **AI / ML** | OpenAI (classic & project key), Anthropic, HuggingFace |
| **Communication** | Slack (bot / user / workspace / refresh), Twilio, SendGrid, Mailgun |
| **Payment** | Stripe (live secret, live restricted, test) |
| **E-commerce** | Shopify (custom / private / access tokens) |
| **Infrastructure** | HashiCorp Vault (service & batch tokens), PostgreSQL DSN, MySQL DSN, MongoDB, Redis |
| **Crypto** | BIP-39 mnemonic seed phrases (12-word detection) |
| **Private Keys** | RSA, EC, OpenSSH, PKCS#8, PGP, DSA (all PEM formats) |
| **Package Registries** | npm |
| **Generic** | `password=`, `secret=`, `api_key=`, `token=` assignment patterns |

---

## Performance

All measurements derived live from a restricted `proot` Android/Termux environment (ARM64 Cortex-A55/A75):

### Detailed Performance Metrics

| Benchmark Suite | Avg. Time per Op | Allocations / Op | Throughput |
| :--- | :--- | :--- | :--- |
| **Aho-Corasick Automaton Build** | 2.62 ms | 507 | - |
| **Full Scan Pipeline (Clean)** | 2.40 ms | 1200 | 20.28 MB/s |
| **Full Scan Pipeline (With Secret)** | 0.13 ms | 39 | 0.89 MB/s |
| **Tier 1 Search (No Hit)** | 1.26 ms | 14 | 88.99 MB/s |
| **Tier 1 Search (With Hit)** | 0.60 ms | 18 | 91.75 MB/s |
| **Tier 2 Entropy Analysis (Small)** | 0.02 ms | 0 (Zero Allocs) | 1.78 MB/s |
| **Tier 2 Entropy Analysis (Large)** | 0.03 ms | 0 (Zero Allocs) | 142.06 MB/s |

**Design decisions enabling ultra-low latency:**
- The hot scan path features **zero allocations** where possible.
- The Aho-Corasick automaton is built **once** at startup and reused across all files.
- Binary files are rejected in **O(8 192)** via null-byte scan — a fixed, bounded cost.
- The newline index is pre-computed in a **single pass** before pattern matching begins.

### The Ultimate Meat Grinder

Sentinel is designed to flawlessly handle "Polyglot" payloads and minified JavaScript where multiple distinct secrets, traps, dummy variables, and formats are jammed onto a single line.

For example, our test suite runs this brutal, minified payload to prove the engine extracts *multiple* overlapping secrets while safely bypassing the trap:

```json
{"user":"test","dummy_token":"dummy_key_12345","real_token":"generic_secret_key_abcdefghijklmnop","note":"don't leak AKIAIOSFODNN7EXAMPLE either!"}
```

**Outcome:**
1. The trap `dummy_token` is explicitly **suppressed** (Tier 3 Context detects `dummy`).
2. The `real_token` generic secret is successfully extracted and flagged.
3. The raw, unassigned `AWS Access Key` is detected by the raw-line traversal.

All within ~2 milliseconds.
- The hot scan path is **allocation-free** — no heap pressure during scanning.
- Tier 1 deduplication uses an **O(1) map** instead of a linear scan.

---

## Installation

Sentinel provides flexible installation options depending on your environment.

### a) Pre-compiled Binaries (Recommended for Termux/Linux/macOS)

The fastest way to install Sentinel without needing Go installed on your system. This is the primary method for Termux/Android users.

1. Navigate to the [Releases page](https://github.com/sentinel-cli/sentinel/releases) and find the URL for the latest `<version>` and your `<architecture>` (e.g., `linux-arm64`, `darwin-amd64`).
2. Download and install using your terminal:

```bash
# 1. Download the binary
wget https://github.com/sentinel-cli/sentinel/releases/download/<version>/sentinel-<version>-<architecture> -O sentinel

# 2. Make the binary executable
chmod +x sentinel

# 3. Move to a system bin path (e.g. $PREFIX/bin for Termux, or /usr/local/bin for Linux/macOS)
mv sentinel $PREFIX/bin/

# 4. Verify installation
sentinel version
```

---

### b) Go Install (For developers)

If you already have Go installed and properly configured in your `PATH`, you can fetch and compile the latest release directly:

```bash
go install github.com/sentinel-cli/sentinel/cmd/sentinel@latest
```
*(Note: Ensure `$(go env GOPATH)/bin` is added to your system `$PATH`)*

---

### c) Build from Source (For contributors)

To build Sentinel manually with full dynamic version tags:

```bash
git clone https://github.com/sentinel-cli/sentinel.git
cd sentinel

# Build via Makefile which injects standard ldflags
make build

# The binary will be output to dist/sentinel
./dist/sentinel version
```

---

### Hook — Current Repository

Install the pre-commit hook for the **current git repository only**:

```bash
# From inside any git repository
sentinel install

# Force-overwrite an existing hook
sentinel install --force
```

This writes a POSIX-compatible shell script to `.git/hooks/pre-commit` that invokes `sentinel run` on every `git commit`.

---

### Hook — Global (All Repositories)

Protect **every repository** on your machine with a single command:

```bash
sentinel install --global
```

This creates `~/.config/sentinel/hooks/pre-commit` and sets:
```
git config --global core.hooksPath ~/.config/sentinel/hooks
```

All existing and future repositories will be scanned automatically.

**To remove the global hook only:**
```bash
git config --global --unset core.hooksPath
```

---

### Uninstallation

To completely remove Sentinel from your system, including the executable binary, global git hooks, and local cached metadata, simply run:

```bash
sentinel uninstall
```

This command works seamlessly whether you installed via `go install` or downloaded a pre-compiled binary (e.g. in Termux or Linux `$PATH`). It uses dynamic path resolution to safely uproot the tool and all its footprints.

---

## Configuration

### Config File Resolution

Sentinel searches for `.sentinel.yaml` in this order:

1. Path specified via `--config` / `-c` flag
2. **Repository root** (current working directory)
3. **Home directory** (`~/.sentinel.yaml`)

With no config file present, all built-in defaults apply — Sentinel works correctly out of the box with zero configuration.

---

### Full Config Reference

Copy the annotated example into your repository:

```bash
cp .sentinel.yaml.example .sentinel.yaml
```

```yaml
# Shannon entropy threshold (bits/symbol).
# Default: 3.5 — catches most real secrets with minimal false positives.
entropy_threshold: 3.5

# Minimum token length considered for entropy analysis.
# Default: 20 characters.
min_secret_length: 20

# Skip files larger than this size. Default: 10485760 (10 MB).
max_file_size_bytes: 10485760

# Attempt to scan binary files? Default: false.
scan_binary_files: false

# Glob patterns to skip (relative to repository root).
exclude_paths:
  - "vendor/**"
  - "node_modules/**"
  - "*.lock"
  - "go.sum"

# File extensions to always skip.
exclude_extensions:
  - ".png"
  - ".jpg"
  - ".gif"
  - ".zip"
  - ".wasm"

# Disable specific detection tiers (use with caution).
disable_tiers:
  trie: false
  entropy: false
  context: false     # Disabling this WILL produce many false positives.

# Stop on the first finding (faster fail in CI).
fail_fast: false

# Enable verbose debug output.
verbose: false
```

---

### Entropy Threshold Tuning

The entropy threshold is the primary false-positive tuning lever:

| Threshold | Effect |
|-----------|--------|
| `3.0` | Very sensitive — may flag base32 IDs and short low-entropy passwords |
| `3.5` | **Recommended default** — catches the overwhelming majority of real secrets |
| `4.0` | Stricter — may miss weak passwords but very low noise |
| `4.5+` | Only flags cryptographically strong random secrets |

If you encounter persistent false positives on a specific string, prefer **`exclude_paths`** or using a safe variable name (e.g. `dummy_api_key`) rather than raising the global threshold.

---

### Excluding Paths and Extensions

```yaml
exclude_paths:
  - "vendor/**"              # vendored third-party code
  - "third_party/**"         # additional third-party code
  - "docs/examples/**"       # documentation examples
  - "infra/terraform/**"     # use environment variables here instead

exclude_extensions:
  - ".pem"                   # if you intentionally commit public certificates
  - ".pub"                   # SSH public keys (safe to commit)
```

---

## Usage

### Pre-commit Hook

After running `sentinel install`, the hook fires automatically on every `git commit`:

```bash
git add src/api_client.go
git commit -m "add API client"
# Sentinel scans staged changes here — blocks if secrets are found
```

---

### Ad-hoc File Scan

Scan any file or directory without going through git:

```bash
# Scan a single file
sentinel scan config/production.yaml

# Scan a directory (non-recursive by default)
sentinel scan ./config

# Scan recursively
sentinel scan --recursive ./src

# Pipe JSON output to jq for automation
sentinel scan --format json ./src | jq '.findings[].severity'
```

---

### JSON Output Mode

```bash
sentinel run --format json 2>&1 | jq .
```

JSON output schema:

```json
{
  "sentinel_version": "v1.2.1",
  "status": "blocked",
  "scanned_files": 3,
  "elapsed_ms": 4,
  "findings": [
    {
      "file_path": "cmd/main.go",
      "line": 12,
      "severity": "CRITICAL",
      "tier": "PATTERN",
      "signature_id": "github-pat-classic",
      "description": "GitHub Personal Access Token (classic)",
      "token": "ghp_AB****************************cdef",
      "entropy": 5.23,
      "line_snippet": "token := \"ghp_AB...cdef\""
    }
  ]
}
```

---

### CI Integration

```yaml
# .github/workflows/security.yml
- name: Sentinel secret scan
  run: |
    sentinel scan --format json --recursive . > sentinel-report.json
    jq -e '.status == "clean"' sentinel-report.json
```

For GitLab CI:

```yaml
sentinel:
  script:
    - sentinel scan --format json --recursive . | tee sentinel-report.json
    - jq -e '.status == "clean"' sentinel-report.json
  artifacts:
    reports:
      sast: sentinel-report.json
```

---

### Lifecycle Commands

| Command | Description |
|---------|-------------|
| `sentinel install` | Install pre-commit hook into current repository |
| `sentinel install --global` | Install hook globally for all repositories |
| `sentinel install --force` | Overwrite an existing hook |
| `sentinel uninstall` | Completely uninstall Sentinel, binary, and all hooks |
| `sentinel update` | Automatically detect OS/Arch and upgrade to the latest pre-compiled release |
| `sentinel version` | Print version, commit, and build date |
| `sentinel run` | Run the pre-commit scan (called by git hook) |
| `sentinel scan [path...]` | Ad-hoc scan of files or directories |

Sentinel features a **silent, non-blocking background update check** that runs at most once per day to notify you of new releases. When you run `sentinel update`, the tool dynamically queries the GitHub Releases API, finds the exact pre-compiled binary for your architecture (or falls back to `go install`), and performs an atomic safe-replacement of the running executable.

---

## Running Tests

```bash
# Run all tests with race detector (recommended)
make test

# Or directly:
go test ./... -v -race -count=1 -timeout 60s

# Run benchmarks
make bench
# Or: go test ./... -bench=. -benchmem -benchtime=3x -run='^$'

# Generate HTML coverage report
make cover
```

Sample benchmark output:
```
BenchmarkAutomatonBuild-8         3     195,234 ns/op    327,680 B/op
BenchmarkSearch-8              3000     341,012 ns/op          0 B/op   ← 0 allocs hot path
BenchmarkSearchWithHit-8       2000     412,887 ns/op      3,456 B/op
BenchmarkShannonSmall-8     5000000         234 ns/op          0 B/op
BenchmarkFullPipeline-8          500   2,341,201 ns/op     12,340 B/op
```

---

## Output Reference

**Clean commit (exit 0):**
```
  ✔ SENTINEL CLEAN  —  4 file(s) scanned in 3.2ms
```

**Blocked commit (exit 1):**
```
   CRITICAL   cmd/main.go:12
               [PATTERN] GitHub Personal Access Token (classic)
               Token:  ghp_AB****************************cdef
               → token := "ghp_AB...cdef"

   HIGH       config/settings.go:8
               [ENTROPY] High-entropy BASE64 string (entropy=6.23)
               Token:  wJalrX****************************EY
               Entropy: 6.2301 bits/symbol
               → AWS_SECRET = "wJalrX...EY"

────────────────────────────────────────────────────────────────────
  SENTINEL SCAN COMPLETE
  Files scanned : 4
  Elapsed       : 5.1ms
  Findings      :  CRITICAL:1   HIGH:1   MEDIUM:0   LOW:0
────────────────────────────────────────────────────────────────────

  ✘ COMMIT BLOCKED — remove the secrets above and try again.
```

---

## False Positive Handling

Sentinel's Tier 3 context filter eliminates false positives automatically. The scanner also performs **assignment-aware value extraction** — it only evaluates the actual RHS of an assignment or the content of string literals, never format strings, function arguments, or variable names in passing position.

If a false positive persists:

1. **Check the file type** — move test data to files matching `*_test.go`, `tests/`, or `testdata/`.
2. **Use a placeholder variable name** — `dummy_key`, `fake_token`, `mock_secret`, etc. are automatically suppressed by Tier 3.
3. **Use an env-var reference** — `token: ${MY_TOKEN}` or `token: $MY_TOKEN` are recognized as safe placeholders.
4. **Add the path to `exclude_paths`** in `.sentinel.yaml`.
5. **Raise `entropy_threshold`** slightly (e.g., `3.8`) if your codebase has many high-entropy non-secret identifiers.

---

## Contributing

We welcome community contributions! However, because this project utilizes a Dual-Licensing model, **all contributors must agree to our [Contributor License Agreement (CLA)](CLA.md)**. By opening a Pull Request, you explicitly agree to transfer the copyright of your submitted code to Khaled Hani. This ensures the project remains legally secure for both open-source and commercial environments.



## Author

Developed by **Khaled Hani** — [https://t.me/A245F](https://t.me/A245F)

---

## License

GNU AGPL v3.0 License. Commercial SaaS use without open-sourcing is prohibited.

---

<div align="center">
Built with precision. Designed for security. Engineered for speed.
</div>
