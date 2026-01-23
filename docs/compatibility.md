# Compatibility Matrix & Acceptance Criteria

This document defines the **Phase 0 compatibility matrix** and the initial acceptance criteria
for LocalAIStack MVP foundation work. It will be expanded as runtimes, drivers, and modules
are implemented.

## Compatibility Matrix (Phase 0)

| Layer | Target | Supported (Phase 0) | Notes |
| --- | --- | --- | --- |
| OS | Ubuntu 22.04 LTS | ✅ | Primary development target |
| OS | Ubuntu 24.04 LTS | ✅ | Primary development target |
| CPU | x86_64 | ✅ | Baseline for CI and developer machines |
| CPU | arm64 | ✅ (build) | Cross-compile supported; runtime validation TBD |
| GPU | NVIDIA (CUDA) | ✅ (planning) | Driver/runtime validation in P1 |
| GPU | CPU-only | ✅ | Should run control plane and lightweight services |
| Driver | NVIDIA 535+ | ✅ (planning) | Exact versions validated in P1 |
| Runtime | Docker | ✅ (planning) | Runtime wiring in P1 |
| Runtime | Native | ✅ (planning) | Minimal commands in P1 |

> ✅ (planning) indicates validated policy/metadata entries are ready, while automated
> validation comes in later phases.

## Acceptance Criteria (Phase 0)

### Build & Test Gate

- `make build` succeeds for server + CLI (current platform).
- `make test` executes unit tests without failure.
- `make test-coverage` enforces the minimum coverage threshold (`COVERAGE_THRESHOLD`).

### Configuration & Observability

- Configuration file + environment overrides are honored in the expected priority order.
- Structured logging is enabled with both JSON and console formats.
- Health check endpoint `/health` returns HTTP 200.

### Control Plane Skeleton

- Control layer initializes hardware detector, policy engine, and state manager stubs
  without crashing.
- API server starts with configuration-driven address/timeouts.

## Validation Checklist

- [ ] Build all binaries via Makefile targets.
- [ ] Run unit tests and coverage gate locally.
- [ ] Verify configuration overrides via env and config file.
- [ ] Confirm `/health` response while server is running.

