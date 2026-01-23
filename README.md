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

## What LocalAIStack Provides

LocalAIStack is not a single application.
It is a **stacked system** composed of coordinated layers.

### 1. System and Environment Management

* Supported operating systems:

  * Ubuntu 22.04 LTS
  * Ubuntu 24.04 LTS
* GPU driver and CUDA compatibility management
* System-level package and mirror configuration
* Safe upgrades and rollback mechanisms

---

### 2. Programming Language Environments (On Demand)

* Python (multiple versions, isolated environments)
* Java (OpenJDK 8 / 11 / 17)
* Node.js (LTS, version-managed)
* Ruby
* PHP
* Rust

All language environments are:

* Optional
* Isolated
* Upgradable
* Removable without system pollution

---

### 3. Local AI Inference Runtimes

Supported inference engines include:

* Ollama
* llama.cpp
* vLLM
* SGLang

Availability is automatically gated by hardware capability (e.g. GPU memory, interconnects).

---

### 4. AI Development Frameworks

* PyTorch
* TensorFlow (optional)
* Hugging Face Transformers
* LangChain
* LangGraph

Framework versions are aligned with installed runtimes and CUDA configurations.

---

### 5. Data and Infrastructure Services

Optional local services for AI development and RAG workflows:

* PostgreSQL
* MySQL
* Redis
* ClickHouse
* Nginx

All services support:

* One-click start/stop
* Persistent data directories
* Local-only or network-accessible modes

---

### 6. AI Applications

Curated open-source AI applications, deployed as managed services:

* RAGFlow
* ComfyUI
* open-deep-research
* (Extensible via manifests)

Each application includes:

* Dependency isolation
* Port management
* Unified access endpoints

---

### 7. Developer Tools

* VS Code (local server mode)
* Aider
* OpenCode
* RooCode

Tools are integrated but not mandatory.

---

### 8. Model Management

LocalAIStack provides a unified model management layer:

* Model sources:

  * Hugging Face
  * ModelScope
* Supported formats:

  * GGUF
  * safetensors
* Capabilities:

  * Search
  * Download
  * Integrity verification
  * Hardware compatibility checks

---

## Hardware Capability Awareness

LocalAIStack classifies hardware into capability tiers and automatically adapts available features.

Example tiers:

* **Tier 1**: Entry-level (≤14B inference)
* **Tier 2**: Mid-range (≈30B inference)
* **Tier 3**: High-end (≥70B, multi-GPU, NVLink)

Users never install software that their hardware cannot reliably run.

---

## User Interface

LocalAIStack provides:

* A web-based management interface
* A CLI for advanced users

### Internationalization

* Built-in multilingual UI support
* Optional AI-assisted interface translation
* No hardcoded language assumptions

---

## Architecture Overview

```
LocalAIStack
├── Control Layer
│   ├── Hardware Detection
│   ├── Capability Policy Engine
│   ├── Package & Version Management
│
├── Runtime Layer
│   ├── Container-based execution
│   ├── Native high-performance paths
│
├── Software Modules
│   ├── Languages
│   ├── Inference Engines
│   ├── Frameworks
│   ├── Services
│   └── Applications
│
└── Interfaces
    ├── Web UI
    └── CLI
```

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
