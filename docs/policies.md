## Hardware Capability and Policy Mapping

---

## 1. Purpose

This document defines how LocalAIStack maps **hardware characteristics to allowed software capabilities**.

Policies act as a safety and predictability layer between hardware detection and software installation.

---

## 2. Policy Design Principles

### 2.1 Declarative Policies

Policies define **constraints and permissions**, not actions.

---

### 2.2 Conservative Defaults

If a capability cannot be reliably supported, it is disabled by default.

---

### 2.3 Explicit Overrides

User overrides are allowed but always explicit and traceable.

---

## 3. Hardware Profile Inputs

Policies consume normalized hardware profiles:

* CPU cores and topology
* System RAM
* GPU count
* GPU VRAM
* GPU interconnects (e.g. NVLink)

---

## 4. Capability Dimensions

Policies operate on the following dimensions:

* Maximum model size
* Allowed inference runtimes
* Parallel execution limits
* Memory and VRAM utilization ceilings

---

## 5. Capability Tiers (Example)

| Tier   | Typical Capability |
| ------ | ------------------ |
| Tier 1 | ≤14B inference     |
| Tier 2 | ≈30B inference     |
| Tier 3 | ≥70B / multi-GPU   |

Tiers are **policy-derived**, not hardcoded.

---

## 6. Policy Definition Example

```yaml
policies:
  - name: tier2-default
    conditions:
      gpu_vram_min: 32GB
      ram_min: 64GB
    allow:
      max_model_size: 30B
      runtimes:
        - llama.cpp
        - vllm
    deny:
      - multi_gpu_training
```

---

## 7. Policy Evaluation Flow

1. Detect hardware
2. Normalize profile
3. Evaluate matching policies
4. Merge allowed capabilities
5. Expose effective capability set

---

## 8. Conflict Resolution

* Most restrictive rule wins
* Explicit denies override allows
* User overrides require confirmation

---

## 9. User Overrides

Overrides are:

* Local-only
* Versioned
* Reversible

Overrides never modify base policy definitions.

---

## 10. Non-Goals

* Policies do not optimize performance
* Policies do not schedule workloads
* Policies do not manage runtime behavior

---

## 11. Summary

Policy mapping ensures that:

* Users are not exposed to unsafe configurations
* Software availability matches hardware reality
* LocalAIStack remains predictable across machines