## Module System and Manifest Specification

---

## 1. Purpose

This document defines the **module system** used by LocalAIStack.

Modules are the fundamental units of installation, execution, and lifecycle management.
Everything managed by LocalAIStack — languages, runtimes, frameworks, services, applications, and tools — is represented as a module.

---

## 2. Design Principles

### 2.1 Modules Are Declarative

Modules describe **what they are and what they require**, not how they are installed.

Imperative logic belongs to the runtime layer, not the module definition.

---

### 2.2 Modules Are Self-Contained

Each module:

* Declares its own dependencies
* Defines its hardware constraints
* Exposes explicit interfaces
* Can be installed, upgraded, or removed independently

---

### 2.3 Modules Are Hardware-Aware but Hardware-Agnostic

Modules may **declare requirements** (e.g. VRAM, CUDA version),
but they do **not hardcode assumptions** about specific GPUs or vendors.

---

## 3. Module Categories

LocalAIStack defines the following module categories:

| Category      | Description                                 |
| ------------- | ------------------------------------------- |
| `language`    | Programming language environments           |
| `runtime`     | AI inference engines and execution backends |
| `framework`   | AI and ML frameworks                        |
| `service`     | Infrastructure services                     |
| `application` | End-user AI applications                    |
| `tool`        | Developer tools                             |
| `model`       | Model metadata and storage descriptors      |

Categories are informational and do not imply execution semantics.

---

## 4. Module Lifecycle States

Each module exists in one of the following states:

* `available`
* `resolved`
* `installed`
* `running`
* `stopped`
* `failed`
* `deprecated`

State transitions are managed exclusively by the Control Layer.

### 4.1 State Machine

Allowed transitions:

| From → To | Notes |
| --- | --- |
| `available` → `resolved` | Dependencies are resolved |
| `resolved` → `installed` | Installation completed |
| `installed` → `running` | Runtime started |
| `installed` → `stopped` | Installed but idle |
| `running` → `stopped` | Runtime stopped |
| `stopped` → `running` | Runtime resumed |
| `*` → `failed` | Error during lifecycle |
| `*` → `deprecated` | Deprecated/removed from registry |

The control layer may also move a failed module back to `resolved` after remediation.

---

## 5. Manifest Overview

Each module is defined by a **manifest file** written in YAML.

Example:

```yaml
name: llama.cpp
category: runtime
version: 0.2.15
description: High-performance local LLM inference engine

hardware:
  gpu:
    vram_min: 8GB
  cpu:
    cores_min: 4

dependencies:
  system:
    - cmake
  runtime:
    - cuda>=12.1

runtime:
  modes:
    - native
    - container

interfaces:
  provides:
    - openai-compatible-api

integrity:
  checksum: sha256:...
```

---

## 6. Manifest Fields

### 6.1 Metadata

```yaml
name: string
category: string
version: string
description: string
license: string (optional)
```

---

### 6.2 Hardware Requirements

```yaml
hardware:
  cpu:
    cores_min: integer
  memory:
    ram_min: string
  gpu:
    vram_min: string
    multi_gpu: boolean
```

If requirements are not met, the module is **not installable**.

---

### 6.3 Dependencies

```yaml
dependencies:
  system:
    - package-name
  modules:
    - module-name
    - module-name@>=1.2.3
  runtime:
    - runtime-constraint
```

Dependencies are resolved before installation planning.

---

### 6.4 Runtime Declaration

```yaml
runtime:
  modes:
    - container
    - native
  preferred: native
```

The Control Layer decides the final execution mode.

---

### 6.5 Interfaces

```yaml
interfaces:
  provides:
    - api
    - service
  consumes:
    - database
```

Interfaces are logical contracts, not network bindings.

---

### 6.6 Integrity Metadata

```yaml
integrity:
  checksum: sha256:...
  signature: <optional detached signature>
```

If a checksum is provided, the registry validates it before accepting the manifest.

---

## 7. Manifest Schema (YAML)

LocalAIStack publishes a machine-readable schema for validation at:

* [`docs/modules.schema.yaml`](./modules.schema.yaml)

This schema is used by the registry validator to ensure manifests are complete and consistent.

---

## 8. Module Registry

The registry is the source of truth for available modules. It:

* Loads manifests from a registry directory
* Validates schema and integrity metadata
* Stores multiple versions per module
* Exposes a version-aware lookup API

Manifests can be grouped by category or vendor inside the registry directory, as long as
each manifest file is discoverable.

---

## 9. Dependency Resolution & Conflict Handling

Dependencies are resolved before installation planning.

* Each module may depend on other modules using optional version constraints.
* The resolver selects the newest compatible version available.
* If a dependency cannot be satisfied, resolution fails with a conflict error.
* Cycles are detected and rejected.

The resolver outputs a topologically sorted installation plan so dependencies are installed
before dependents.

---

## 10. Versioning and Compatibility

* Modules are versioned independently
* Compatibility is validated at resolution time
* Breaking changes must increment major versions

---

## 11. Extension Model

New modules can be added by:

* Adding a manifest file
* Registering it with the module registry

No core code modification is required.

---

## 12. Non-Goals

* Modules do not embed secrets
* Modules do not perform installation logic
* Modules do not mutate global state directly

---

## 13. Summary

The module system ensures LocalAIStack remains:

* Extensible
* Predictable
* Hardware-aware
* Long-term maintainable
