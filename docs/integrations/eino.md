# Eino Integration Plan for LocalAIStack

## 1. Goals and Scope

This plan integrates the Eino framework as a first-class **framework module** in LocalAIStack, aligning with the module system, control layer, and runtime execution model.
The initial integration focuses on:

- shipping Eino as a versioned, installable module
- wiring module metadata into the control layer’s resolver/state tracking
- exposing Eino’s availability via CLI/API
- preparing runtime hooks for optional Eino DevOps tooling (if adopted later)

## 2. Module Placement and Directory Layout

Create a new module at:

```
modules/eino/
├── manifest.yaml
├── INSTALL.yaml
├── scripts/
│   ├── verify.sh
│   ├── rollback.sh
│   ├── uninstall.sh
│   ├── purge.sh
│   └── cleanup_soft.sh (optional)
└── templates/ (optional)
```

This layout follows the required InstallSpec structure and keeps Eino aligned with existing module conventions.

## 3. Module Manifest Details (modules/eino/manifest.yaml)

**Category:** `framework`

Recommended fields (aligned with `internal/module/types.go` and `docs/modules.md`):

```yaml
name: eino
category: framework
version: <eino-version>
description: Golang LLM application framework (CloudWeGo Eino)
license: Apache-2.0

hardware: {}

dependencies:
  system:
    - git
  modules:
    - go

runtime:
  modes:
    - native
  preferred: native

interfaces:
  provides:
    - llm-orchestration
    - workflow-runtime
```

Notes:
- **`modules: [go]`** assumes a Go language module exists or will be added under `modules/go/`.
- Eino is a **library/framework** (no long-lived service required), so runtime is `native` and installation is focused on installing sources/binaries or vendoring the Go module.

## 4. InstallSpec (modules/eino/INSTALL.yaml)

Key ideas for InstallSpec:

- **Install mode:** `native`
- **Install actions:**
  - ensure Go toolchain is present (module dependency)
  - clone or `go env GOPATH`-based install into a LocalAIStack-managed cache
  - optionally provide a pinned version via `go env GOPATH` + `GOMODCACHE` or a vendor directory under `/var/lib/localaistack/frameworks/eino`
- **Verification:**
  - `go list` on `github.com/cloudwego/eino/...`
  - optional `go test` for a minimal package subset
- **Rollback/Uninstall:**
  - remove cached module directories and workspace entries

## 5. Control Layer Integration

Tie the Eino module into the control layer by extending the module registry/resolver:

- **Module registry loading** should include `modules/eino/manifest.yaml` and use `internal/module/Manifest` definitions.
- **Capability exposure**: treat Eino as a `framework` that provides `llm-orchestration` capability so the resolver can satisfy dependencies for future “app” modules.
- **State tracking**: add Eino into `StateManager` module map and ensure install/upgrade transitions are tracked.

This keeps the control plane consistent with `docs/architecture.md` and `docs/modules.md` expectations.

## 6. Runtime Layer Integration

Eino is library-centric (no default daemon), so runtime integration is minimal:

- **No service lifecycle required** by default.
- If future Eino DevOps tooling is added (e.g., web UI or tracing service), introduce a separate `application` or `service` module (e.g., `modules/eino-devops/`) with `container` runtime mode.

## 7. CLI / API Exposure

Expose Eino in existing module commands:

- `localaistack module list` should include `eino` as available.
- `localaistack module install eino` should route to the resolver and install plan for the Eino module.

For API:

- Add module listing endpoints to surface `framework` modules and their states.

## 8. Suggested Next Steps (Implementation Order)

1. **Create `modules/eino/`** with `manifest.yaml` + `INSTALL.yaml` + scripts.
2. **Implement module registry loading** in the control layer to discover `modules/*/manifest.yaml`.
3. **Wire resolver + state manager** to accept `framework` modules and track Eino state transitions.
4. **Expose module states in CLI/API** for visibility and future UX.
5. **Optional:** add `modules/eino-ext/` for EinoExt tooling/examples as a separate module.
