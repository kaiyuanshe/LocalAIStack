## Runtime Execution Model

---

## 1. Purpose

This document describes how LocalAIStack executes software modules.

It explains the rationale and boundaries between **container-based** and **native execution** modes.

---

## 2. Runtime Design Principles

### 2.1 Execution Is Separated from Policy

Runtime components execute instructions provided by the Control Layer.

They do not make decisions.

---

### 2.2 Performance Where Required, Isolation Where Possible

Isolation is preferred by default.
Native execution is reserved for performance-critical paths.

---

## 3. Runtime Components

LocalAIStack implements a runtime manager with two execution backends:

* **Container runtime**: Docker or Podman CLI integration.
* **Native runtime**: Direct process execution on the host.

The manager tracks process state, captures logs, and publishes health status for every running module.

---

## 4. Execution Modes

### 4.1 Container-Based Execution (Default)

Used for:

* Services
* Applications
* Developer tools
* Non-performance-critical components

**Advantages**

* Isolation
* Reproducibility
* Easier upgrades and rollbacks

---

### 4.2 Native Execution

Used for:

* llama.cpp
* vLLM (high-throughput paths)
* CUDA-sensitive workloads

**Advantages**

* Maximum performance
* Direct hardware access

---

## 5. Mode Selection Strategy

Execution mode is determined by:

1. Module manifest declaration (`runtime.modes` + optional `runtime.preferred`)
2. Policy constraints (allowed runtimes)
3. Local runtime configuration (`runtime.default_mode`, `runtime.docker_enabled`, `runtime.native_enabled`)
4. User preference (optional override)

If the preferred mode is unavailable, the runtime manager falls back to the default mode or the first available mode.

---

## 6. Runtime Responsibilities

* Process lifecycle
* Resource allocation
* Log capture
* Health reporting

---

## 7. Process Lifecycle & Log Collection

The runtime manager supports:

* **Start/Stop**: launch and terminate module processes or containers.
* **Monitoring**: track running state and exit status.
* **Log capture**: stream stdout/stderr to per-module log files under `runtime.log_dir`.

Container logs are collected via `docker logs`/`podman logs`.
Native processes stream logs directly from stdout/stderr.

---

## 8. Health Reporting

Health status is reported as:

* **healthy**: process/container is running and optional checks succeed.
* **unhealthy**: process/container has exited or health checks fail.
* **unknown**: no health signal yet.

For containers with health checks configured in the image, the runtime manager reads the container health status.
For native processes, the manager reports healthy while the process is running or executes an optional health command.

---

## 9. Runtime Non-Responsibilities

* No dependency resolution
* No policy evaluation
* No UI logic

---

## 10. Resource Management

* GPU access is explicit
* Memory limits are enforced where possible
* Overcommitment is avoided by policy

---

## 11. Failure Handling

Runtime failures result in:

* Explicit error states
* Preserved logs
* No silent retries unless configured

---

## 12. Security Boundaries

* Containers run with minimal privileges
* Native execution is limited to trusted modules
* No implicit network exposure

---

## 13. Future Evolution

Potential extensions:

* Alternative container backends
* Hybrid execution modes
* Multi-node runtimes (optional)

---

## 14. Summary

The runtime model balances:

* Safety and isolation
* Performance and control
* Predictability and flexibility

LocalAIStack treats execution as infrastructure, not automation.

---

## 15. Related Docs

For smart model run tuning and operational flags, see:

* [`docs/smart_run_p2.md`](./smart_run_p2.md)
