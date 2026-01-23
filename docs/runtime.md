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

## 3. Execution Modes

### 3.1 Container-Based Execution (Default)

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

### 3.2 Native Execution

Used for:

* llama.cpp
* vLLM (high-throughput paths)
* CUDA-sensitive workloads

**Advantages**

* Maximum performance
* Direct hardware access

---

## 4. Mode Selection Strategy

Execution mode is determined by:

1. Module manifest declaration
2. Hardware capability
3. Policy constraints
4. User preference (optional)

---

## 5. Runtime Responsibilities

* Process lifecycle
* Resource allocation
* Log capture
* Health reporting

---

## 6. Runtime Non-Responsibilities

* No dependency resolution
* No policy evaluation
* No UI logic

---

## 7. Resource Management

* GPU access is explicit
* Memory limits are enforced where possible
* Overcommitment is avoided by policy

---

## 8. Failure Handling

Runtime failures result in:

* Explicit error states
* Preserved logs
* No silent retries unless configured

---

## 9. Security Boundaries

* Containers run with minimal privileges
* Native execution is limited to trusted modules
* No implicit network exposure

---

## 10. Future Evolution

Potential extensions:

* Alternative container backends
* Hybrid execution modes
* Multi-node runtimes (optional)

---

## 11. Summary

The runtime model balances:

* Safety and isolation
* Performance and control
* Predictability and flexibility

LocalAIStack treats execution as infrastructure, not automation.