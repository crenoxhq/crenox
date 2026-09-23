## Summary of Changes

<!-- Provide a concise technical summary of the modifications introduced by this pull request. -->

## Motivation and Context

<!-- Explain why this change is necessary and what problem it solves. -->

## Related Issues

<!-- Link the relevant issue(s) addressed by this PR. For example: Closes #123 -->
Closes # (or N/A — standalone change, with explanation)

## Category of Change

- [ ] Bug Fix (resolution of an issue, false positive, or false negative)
- [ ] New Signature (addition of a new credential or secret pattern detector)
- [ ] Performance Optimization (reduction in execution latency or memory allocations)
- [ ] Security Hardening (safeguard, credential masking, or validation)
- [ ] Documentation (updates to website, manual, or code comments)
- [ ] Test Suite (addition or refinement of regression tests and benchmarks)

## Technical Architecture Impact

<!-- Describe the algorithmic or structural impact of the change.
If modifying internal/trie (Aho-Corasick), internal/entropy (Shannon), or internal/context,
confirm that hot-path heap allocations remain at zero (0 B/op, 0 allocs/op). -->

## Verification and Benchmarks

- [ ] `go test -v ./...` executed with zero failures
- [ ] `go test -race ./...` executed with zero data race warnings
- [ ] `go test -bench=. ./...` executed to verify performance regressions
- [ ] Regression test cases added for new behavior or detection patterns

```bash
# Paste benchmark output or test results below:
go test -v ./...
```

## Contributor Checklist

- [ ] My code adheres to the project's Go style conventions (`gofmt`, `go vet`).
- [ ] My code adheres to the zero-allocation hot-path guidelines of this project.
- [ ] Documentation (`README.md`, `CHANGELOG.md`) has been updated where applicable.
- [ ] No active production credentials or sensitive test fixtures are committed in plain text.
- [ ] By submitting this pull request, I confirm that I agree to the Crenox Contributor License Agreement ([CLA.md](CLA.md)) and acknowledge that CrenoxHQ maintains dual-licensing rights for community and enterprise distributions.
