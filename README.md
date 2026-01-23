# LocalAIStack

[中文说明](README.cn.md)

**LocalAIStack** is an open, modular software stack for building and operating **local AI workstations**.

It provides a unified control layer to install, manage, upgrade, and operate AI development environments, inference runtimes, models, and applications on local hardware — without relying on cloud services or vendor-specific platforms.

LocalAIStack is designed to be **hardware-aware**, **reproducible**, and **extensible**, serving as a long-term foundation for local AI computing.

---

## Why LocalAIStack

Running AI workloads locally is no longer a niche use case.
However, the local AI software ecosystem remains fragmented:

* Inference engines, frameworks, and applications evolve independently
* CUDA, drivers, Python, and system dependencies are tightly coupled
* Installation paths vary across hardware configurations
* Environment drift makes reproduction and maintenance difficult
* Many tools assume cloud-first deployment models

LocalAIStack addresses these issues by treating the **local AI workstation itself as infrastructure**.

---

## Design Goals

LocalAIStack is built around the following principles:

* **Local-first**
  No mandatory cloud dependency. Works fully offline if required.

* **Hardware-aware**
  Automatically adapts available software capabilities to CPU, GPU, memory, and interconnects.

* **Modular and composable**
  All components are optional and independently managed.

* **Reproducible by default**
  Installation and runtime behavior are deterministic and version-controlled.

* **Open and vendor-neutral**
  No lock-in to specific hardware vendors, models, or frameworks.

---

## Documentation

The detailed technical design now lives in the `docs/` directory to keep the README concise and consistent.

* [Architecture](./docs/architecture.md)
* [Module System and Manifest Specification](./docs/modules.md)
* [Hardware Capability and Policy Mapping](./docs/policies.md)
* [Runtime Execution Model](./docs/runtime.md)

---

## What LocalAIStack Provides

LocalAIStack is a layered system for managing local AI workstations end to end. At a high level it delivers:

* Deterministic installation, upgrades, and rollbacks for local AI environments
* Hardware-aware runtime selection and policy gating
* Modular components that can be enabled or removed independently
* Unified interfaces for managing runtimes, services, and applications

---

## Typical Use Cases

* Local LLM inference and experimentation
* RAG and agent development
* AI education and teaching labs
* Research reproducibility
* Enterprise private AI environments
* Hardware evaluation and benchmarking

---

## Open Source

LocalAIStack is an open-source project.

* License: Apache 2.0 (or MIT, TBD)
* Contributions are welcome
* Vendor-neutral by design

---

## Project Status

LocalAIStack is under active development.

The initial focus is:

* Stable Tier 2 (≈30B) local inference workflows
* Deterministic installation paths
* Clear hardware-to-capability mapping

Roadmaps and milestones will be published as the project evolves.

---

## Getting Started

Documentation, installation guides, and manifests are available in the `docs/` directory.

---

## Philosophy

LocalAIStack treats **local AI computing as infrastructure**, not as a collection of tools.

It aims to make local AI systems:

* Predictable
* Maintainable
* Understandable
* Long-lived
