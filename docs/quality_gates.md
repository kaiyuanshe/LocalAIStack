# Test & Quality Gates

Phase 0 introduces the baseline testing framework and quality gates.

## Test Types

- **Unit tests**: colocated with packages (`*_test.go`).
- **Integration tests**: reserved for `internal/...` once runtimes and services exist.

## Quality Gates

| Gate | Tooling | Threshold |
| --- | --- | --- |
| Unit test pass | `make test` | Must succeed |
| Coverage gate | `make test-coverage` | `COVERAGE_THRESHOLD` (default 40%) |
| Static checks | `make vet` | Must succeed |
| Formatting | `make fmt` | Must succeed |

## Coverage Gate Behavior

`make test-coverage` generates `coverage.out` and `coverage.html`, then fails if total
coverage drops below the threshold defined in the Makefile.

## CI/CD Expectations

The CI workflow should, at minimum:

1. Run `make test`
2. Run `make test-coverage`
3. Run `make vet`
4. Run `make build`

