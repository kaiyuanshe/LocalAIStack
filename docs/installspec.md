# InstallSpec

## LocalAIStack Installation Specification

---

## 0. Status

* **Version:** v0.1.2
* **Status:** Stable
* **Reference Module:** `modules/ollama/`

---

## 1. Purpose

InstallSpec defines a **machine-first, declarative specification** for describing the full lifecycle of software modules managed by LocalAIStack.

It standardizes how software is:

* discovered
* installed
* configured
* verified
* rolled back
* uninstalled
* fully removed (purged)
* reconciled in non-empty environments

InstallSpec v0.1.2 is designed to be consumed **directly** by:

* automated agents
* LLM-based orchestrators
* CI validation pipelines
* LocalAIStack CLI and UI

---

## 2. Scope

InstallSpec applies to **all LocalAIStack modules**, including:

* programming language runtimes
* AI inference engines
* AI frameworks
* infrastructure services
* AI applications
* developer tools

Any module without a valid `INSTALL.yaml` conforming to this specification is **non-compliant** and MUST NOT be merged.

---

## 3. Design Principles

### 3.1 Machine-First Specification

`INSTALL.yaml` is the **authoritative execution plan**, not supplementary documentation.

Human-readable explanations are allowed only as structured fields and MUST NOT affect execution semantics.

---

### 3.2 Declarative and Deterministic

InstallSpec describes **desired state and allowed transitions**, not ad-hoc scripts.

Given the same inputs (hardware, OS, policy), execution MUST be deterministic.

---

### 3.3 Dependency-Aware Orchestration

Modules declare explicit dependencies, enabling:

* safe install ordering
* correct uninstall and purge ordering
* intelligent substitution based on capabilities

---

### 3.4 Safe by Default

Destructive actions are:

* explicitly named
* isolated
* never implicit
* never auto-selected without confirmation

---

## 4. Required Module Structure

Every module MUST follow this structure:

```text
modules/<module-name>/
├── manifest.yaml              # Module metadata & dependency graph (REQUIRED)
├── INSTALL.yaml               # InstallSpec execution plan (REQUIRED)
├── scripts/
│   ├── verify.sh              # Post-install verification (REQUIRED)
│   ├── rollback.sh            # Failure recovery (REQUIRED)
│   ├── uninstall.sh           # Remove software, keep data (REQUIRED)
│   ├── purge.sh               # Full removal (REQUIRED)
│   ├── cleanup_soft.sh        # Non-destructive cleanup (OPTIONAL*)
│   └── cleanup_full.sh        # Destructive cleanup (OPTIONAL*)
└── templates/                 # Configuration templates (OPTIONAL)
```

* Cleanup scripts become REQUIRED if corresponding rebuild modes are declared.

---

## 5. `manifest.yaml` Responsibilities

`manifest.yaml` is the **machine truth source** for:

* module identity
* supported platforms
* hardware constraints
* dependency graph
* capability contracts

### 5.1 Dependency Model (Mandatory)

InstallSpec defines **four dependency classes**:

```yaml
dependencies:
  system:        # OS-level packages or kernel capabilities
    - curl
    - ca-certificates

  modules:       # Strong LocalAIStack module dependencies (DAG)
    - postgresql
    - cuda-runtime

  capabilities:  # Abstract functional requirements
    - sql_database
    - local_llm_inference

  optional:      # Soft / performance-related dependencies
    gpu:
      - cuda-runtime
```

#### Dependency Semantics

| Class        | Semantics                                          |
| ------------ | -------------------------------------------------- |
| system       | Presence required, not removed on uninstall        |
| modules      | Strong dependency, defines install/uninstall order |
| capabilities | Satisfied by any compatible provider               |
| optional     | Never blocks installation                          |

Circular `modules` dependencies are **forbidden**.

---

## 6. INSTALL.yaml — Top-Level Structure

Every `INSTALL.yaml` MUST conform to the following top-level structure.

```yaml
apiVersion: las.installspec/v0.1.2
kind: InstallPlan

id: <module-id>
category: <module-category>

supported_platforms: [...]
install_modes: [...]
rebuild_modes: [...]

tools_required: [...]

description: ...
dependencies: ...
preconditions: ...
decision_matrix: ...
environment_rebuild: ...
install: ...
configuration: ...
verification: ...
rollback: ...
uninstall: ...
purge: ...
security: ...
```

All sections are REQUIRED unless explicitly marked optional.

---

## 7. Metadata and Description

```yaml
description:
  purpose: string
  scope: [ ... ]
  non_goals: [ ... ]
```

These fields are **informational only** and MUST NOT contain executable logic.

---

## 8. Preconditions

Preconditions define **hard gates** that MUST pass before installation.

Each precondition MUST include:

```yaml
- id: string
  intent: string
  tool: string
  command: string | object
  expected: object
```

If any precondition fails, execution MUST halt.

---

## 9. Decision Matrix

The decision matrix selects an installation path.

```yaml
decision_matrix:
  default: native
  rules:
    - when:
        shared_environment: true
      use: container
```

Decision rules MUST be explicit and side-effect free.

---

## 10. Environment Cleanup and Rebuild

InstallSpec v0.1.2 formally defines **environment reconciliation**.

```yaml
rebuild_modes:
  - none
  - soft
  - full
```

### 10.1 Rebuild Mode Semantics

| Mode | Meaning                                    |
| ---- | ------------------------------------------ |
| none | No cleanup; conflicts cause failure        |
| soft | Remove known conflicts, preserve user data |
| full | Complete cleanup and rebuild               |

---

### 10.2 Cleanup Definition

```yaml
environment_rebuild:
  detect: [ ... ]
  soft_cleanup: [ ... ]   # optional
  full_cleanup: [ ... ]   # optional
```

Cleanup actions follow the same execution rules as install steps.

---

## 11. Installation Procedure

Installation steps are defined per install mode.

```yaml
install:
  native:
    - id: S10
      intent: string
      tool: string
      command: string | object
      expected: object
      idempotent: boolean
```

### 11.1 Action Block Requirements

Each action block MUST include:

* `id`
* `intent`
* `tool`
* `command` or `edit`
* `expected`
* `idempotent`

Optional:

* `on_fail`
* `timeout`

---

## 12. Configuration

```yaml
configuration:
  required: boolean
  defaults: object
  templates: [ ... ]     # optional
```

Secrets MUST NOT be embedded.

---

## 13. Verification

Verification is **mandatory**.

```yaml
verification:
  script: scripts/verify.sh
```

Verification MUST be:

* machine-executable
* deterministic
* sufficient to prove correct installation

---

## 14. Rollback (Failure Recovery)

Rollback is ONLY for failed installs or upgrades.

```yaml
rollback:
  script: scripts/rollback.sh
```

Rollback MUST restore the pre-install state and preserve user data unless documented otherwise.

---

## 15. Uninstall (Keep User Data)

```yaml
uninstall:
  script: scripts/uninstall.sh
  preserves:
    - <paths>
```

Uninstall removes software and services but MUST preserve user data.

---

## 16. Full Removal / Purge

```yaml
purge:
  script: scripts/purge.sh
  destructive: true
```

Purge is:

* explicit
* irreversible
* destructive

It MUST never be auto-selected.

---

## 17. Security Notes

```yaml
security:
  network:
    bind: localhost | 0.0.0.0
    auth: none | required
  privileges:
    requires_sudo: boolean
```

This section MUST document all security-relevant behavior.

---

## 18. Execution Order Guarantees

InstallSpec mandates:

* **Install order:**
  `system → modules → capabilities → optional → install steps`
* **Uninstall / purge order:**
  Reverse topological order of module dependencies

---

## 19. Compliance and Validation

A module is **InstallSpec v0.1.2 compliant** if and only if:

* `INSTALL.yaml` exists and validates
* `manifest.yaml` dependencies are valid
* all required scripts exist
* rebuild semantics are honored
* no forbidden constructs are present

Compliance SHOULD be enforced via CI using schema validation.

---

## 20. Versioning and Evolution

* InstallSpec versions are explicit
* Minor versions MAY add constraints
* Major versions MAY introduce breaking changes
* Backward compatibility is NOT guaranteed across major versions

---

## 21. Summary

InstallSpec v0.1.2 completes the transition from:

> “installation documentation”

to:

> **“agent-executable infrastructure specification”**

By adopting `INSTALL.yaml`, LocalAIStack establishes a foundation for:

* deterministic automation
* dependency-safe orchestration
* LLM-native execution
* long-term maintainability