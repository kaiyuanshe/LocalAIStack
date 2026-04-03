# LocalAIStack

[中文版 README](README.md)

**LocalAIStack** is an open, modular software stack for building and operating a **local AI workstation**.

It provides a unified control layer for installing, managing, upgrading, and running AI development environments, inference runtimes, models, and applications on local hardware, without depending on cloud services or proprietary vendor platforms.

LocalAIStack is designed to be **hardware-aware**, **reproducible**, and **extensible**, serving as a long-term foundation for local AI computing.

## Why LocalAIStack

Running AI workloads locally is no longer a niche requirement, but the local AI software ecosystem is still highly fragmented:

* Inference engines, frameworks, and applications evolve independently
* CUDA, drivers, Python, and system dependencies are tightly coupled
* Installation paths vary across different hardware configurations
* Environment drift makes systems hard to reproduce and maintain
* Many tools assume a cloud-first deployment model

LocalAIStack addresses this by treating the **local AI workstation itself as infrastructure**.

## Design Goals

LocalAIStack is built around the following principles:

* **Local-first**: no mandatory cloud dependency, with offline operation possible when needed
* **Hardware-aware**: automatically adapts software capabilities based on CPU, GPU, memory, and interconnect capacity
* **Modular and composable**: every component is optional and independently manageable
* **Reproducible by default**: installation and runtime behavior is deterministic and versionable
* **Open and vendor-neutral**: no lock-in to a specific hardware vendor, model, or framework

## What LocalAIStack Provides

LocalAIStack is not a single application. It is a layered stack made of cooperating subsystems.

### 1. System and Environment Management

* Supported operating systems:
  * Ubuntu 22.04 LTS
  * Ubuntu 24.04 LTS
* GPU driver and CUDA compatibility management
* System package management and mirror configuration
* Safe upgrade and rollback mechanisms

### 2. Programming Language Environments

Available on demand:

* Python (multiple versions, isolated environments)
* Java (OpenJDK 8 / 11 / 17)
* Node.js (LTS, version-managed)
* Ruby
* PHP
* Rust

All language environments are optional, isolated, upgradable, and removable without polluting the host system.

### 3. Local AI Inference Runtimes

Supported inference engines include:

* Ollama
* llama.cpp
* vLLM
* SGLang

Availability is constrained automatically by hardware capability. For example, when installing `llama.cpp`, the system should evaluate whether a GPU is available and what model it is; if no GPU is present, it should install a CPU-only variant.

### 4. AI Development Frameworks

* PyTorch
* TensorFlow (optional)
* Hugging Face Transformers
* LangChain
* LangGraph
* Unsloth (for local LLM fine-tuning and reinforcement learning)

Framework versions are aligned with installed runtimes and CUDA configuration. This repository already includes an `unsloth` module, installed by default via the official Linux path `python3 -m pip install --user unsloth`, and it requires `Python < 3.14`.

### 5. Data and Infrastructure Services

Optional local services for AI development and RAG workflows:

* PostgreSQL
* MySQL
* Redis
* ClickHouse
* Nginx

All services support:

* One-command start and stop
* Persistent data directories
* Local-only or network-accessible modes

### 6. AI Applications

Curated open-source AI applications deployed as managed services:

* RAGFlow
* ComfyUI
* open-deep-research
* Extendable through manifests

Each application includes:

* Dependency isolation
* Port management
* Unified access entry points

### 7. Developer Tools

* VS Code (local server mode)
* Aider
* OpenCode
* RooCode

These tools are integrated but not mandatory.

### 8. Model Management

LocalAIStack provides a unified model management layer:

* Model sources:
  * Hugging Face
  * ModelScope
  * Ollama
* Supported formats:
  * GGUF
  * safetensors
* Capabilities:
  * Search
  * Download
  * Integrity verification
  * Hardware compatibility checks
  * Repair of missing tokenizer/config support files

## Hardware Capability Awareness

LocalAIStack classifies hardware into capability tiers and adapts available functionality automatically.

Example tiers:

* **Tier 1**: entry level (`<=14B` inference)
* **Tier 2**: mid-range (around `30B` inference)
* **Tier 3**: high-end (`>=70B`, multi-GPU, NVLink)

The system should avoid installing software that the hardware cannot run reliably.

## User Interfaces

LocalAIStack provides:

* A web-based management interface
* A CLI for advanced users

### Internationalization

* Built-in multilingual UI support
* Optional AI-assisted interface translation
* No hard-coded language assumptions

## Architecture Overview

```text
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

## Typical Use Cases

* Local LLM inference and experimentation
* RAG and agent development
* AI education and teaching labs
* Research reproducibility
* Private enterprise AI environments
* Hardware evaluation and benchmarking

## Project Status

LocalAIStack is under active development.

Current early priorities:

* A stable Tier 2 (around `30B`) local inference workflow
* Deterministic installation paths
* Clear hardware-to-capability mapping

Roadmaps and milestones will be published as the project evolves.

## Quick Start

Minimal workflow: clone the repository, build it, then use the built CLI.

Command conventions:

* Linux/macOS: `./build/las` and `./build/las-server`
* Windows PowerShell: `.\build\las.exe` and `.\build\las-server.exe`

### 1. Clone the Repository

```bash
git clone <repo-url> LocalAIStack
cd LocalAIStack
```

### 2. Build

```bash
make tidy
make build
```

Build outputs:

* Linux/macOS: `./build/las` (CLI), `./build/las-server` (server)
* Windows PowerShell: `.\build\las.exe` (CLI), `.\build\las-server.exe` (server)

### 3. Use the CLI

Examples below use Linux/macOS syntax by default; in Windows PowerShell, replace `./build/las` with `.\build\las.exe`.

Start with the overview:

```bash
./build/las --help
```

#### 3.1 Initialization and System Info

```bash
# Interactive initialization (recommended)
./build/las init

# Non-interactive initialization (scriptable)
./build/las init \
  --api-key "sk-xxxx" \
  --language zh-CN \
  --assistant-provider siliconflow \
  --assistant-model deepseek-ai/DeepSeek-V3.2 \
  --translation-provider siliconflow \
  --translation-model tencent/Hunyuan-MT-7B

# System info and hardware detection
./build/las system info
./build/las system detect

# system init is equivalent to init
./build/las system init
```

Effects:

* Generates the config file: `$HOME/.localaistack/config.yaml`
* Default split responsibilities between two models:
  * Translation model: `tencent/Hunyuan-MT-7B`
  * Assistant model: `deepseek-ai/DeepSeek-V3.2` (customizable)
* Generates baseline hardware info in `base_info.json` for the install planner, config planner, and smart-run

#### 3.2 Module Management (`module`)

```bash
# List manageable modules
./build/las module list

# Install / upgrade / uninstall / purge
./build/las module install ollama
./build/las module update ollama
./build/las module uninstall ollama
./build/las module purge ollama

# Health check
./build/las module check ollama

# For example, install the Unsloth training / fine-tuning framework
./build/las module install unsloth
./build/las module check unsloth

# Module-specific settings
./build/las module setting comfyui Comfy-Org_z_image_turbo
```

Modules already integrated in this repository include inference runtimes such as `ollama`, `llama.cpp`, and `vllm`, model tools such as `hf` and `modelscope`, and the `unsloth` training / fine-tuning framework.

The repository also now includes an `obeaver` module, installable with `./build/las module install obeaver`, which provisions the upstream [microsoft/obeaver](https://github.com/microsoft/obeaver) project. On Windows it automatically checks for and installs Foundry Local with `winget install Microsoft.FoundryLocal`; on macOS it automatically checks for and installs Foundry Local with `brew install microsoft/foundrylocal/foundrylocal`. Linux does not support Foundry Local, so use `obeaver run --engine ort <local-onnx-model-dir>` there.

#### 3.3 Post-install Config Planning (`module config-plan`)

```bash
# Generate a config plan for a module (text)
./build/las module config-plan llama.cpp

# JSON output
./build/las module config-plan llama.cpp --output json

# Demonstration only, do not write to disk
./build/las module config-plan vllm --dry-run --planner-debug

# Strict mode: fail immediately if the planner fails
./build/las module config-plan vllm --planner-strict --dry-run

# Save to ~/.localaistack/config-plans/<module>.json
./build/las module config-plan ollama --apply
```

#### 3.4 Model Management (`model`)

```bash
# Search models (supports all / ollama / huggingface / modelscope)
export HF_ENDPOINT=https://hf-mirror.com
./build/las model search qwen3
./build/las model search qwen3 --source huggingface --limit 20

# Download models
./build/las model download qwen3-coder:30b
./build/las model download unsloth/Qwen3-Coder-Next-GGUF
./build/las model download unsloth/Qwen3-Coder-Next-GGUF --file Q4_K_M.gguf

# List downloaded models
./build/las model list

# Repair missing tokenizer/config files
./build/las model repair ByteDance/Ouro-2.6B-Thinking

# Remove a model
./build/las model rm qwen3-coder:30b --force
```

#### 3.5 Running Models (`model run`)

```bash
# Run GGUF (llama.cpp)
./build/las model run unsloth/Qwen3-Coder-Next-GGUF

# Specify runtime parameters
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --ctx-size 65536 \
  --threads 16 \
  --n-gpu-layers 40

# Automatic batch tuning + print command only
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --auto-batch --dry-run

# smart-run (use an LLM to recommend parameters)
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --smart-run --smart-run-debug --dry-run

# Force refresh smart-run (ignore local cache and ask the LLM again)
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --smart-run --smart-run-refresh --smart-run-debug --dry-run

# Inspect smart-run cache
./build/las model smart-run-cache list
./build/las model smart-run-cache list unsloth/Qwen3-Coder-Next-GGUF

# Remove smart-run cache for a model
./build/las model smart-run-cache rm unsloth/Qwen3-Coder-Next-GGUF

# Strict smart-run mode
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --smart-run --smart-run-strict --dry-run

# Run safetensors (vLLM)
./build/las model run ByteDance/Ouro-2.6B-Thinking \
  --vllm-max-model-len 8192 \
  --vllm-gpu-memory-utilization 0.9
```

#### 3.6 Provider and Service Management

```bash
# List available LLM providers
./build/las provider list

# Service management
./build/las service start ollama
./build/las service status ollama
./build/las service stop ollama
```

#### 3.7 Failure Handling Loop (`failure`)

```bash
# List recent failures
./build/las failure list --limit 20

# Filter by phase or category
./build/las failure list --phase smart_run --category timeout

# JSON output
./build/las failure list --output json --limit 5

# Show details and repair suggestions for one failure
./build/las failure show <event-id>
```

Optional debug switch:

```bash
export LOCALAISTACK_FAILURE_DEBUG=1
```

On failure it will additionally print: `phase/category/retryable/log_path/suggestion`.

#### 3.8 Shell Completion

```bash
# Bash
./build/las completion bash > /tmp/las.bash

# Zsh
./build/las completion zsh > /tmp/_las

# Fish
./build/las completion fish > /tmp/las.fish

# PowerShell
./build/las completion powershell > /tmp/las.ps1
```

#### 3.9 Global Flags

```bash
# Use a specific config file
./build/las --config /tmp/las-config.yaml module list

# Enable verbose output
./build/las --verbose model list
```

#### 3.10 Command Tree

The following tree matches the command entry points currently registered in the CLI:

```text
las
├─ init
├─ module
│  ├─ list
│  ├─ install <module>
│  ├─ update <module>
│  ├─ uninstall <module>
│  ├─ purge <module>
│  ├─ check <module>
│  ├─ setting <module> <setting-args...>
│  └─ config-plan <module>
├─ service
│  ├─ start <service>
│  ├─ stop <service>
│  └─ status <service>
├─ model
│  ├─ search <query>
│  ├─ download <model-id> [file]
│  ├─ list
│  ├─ run <model-id> [gguf-file-or-quant]
│  ├─ rm <model-id>
│  ├─ repair <model-id>
│  └─ smart-run-cache
│     ├─ list [model-id]
│     └─ rm <model-id>
├─ provider
│  └─ list
├─ failure
│  ├─ list
│  └─ show <event-id>
└─ system
   ├─ init
   ├─ detect
   └─ info
```

Notes:

* `init` is registered both at the root and as `system init`; they are equivalent
* `module` also has the alias `modules`
* `model repair` also has the alias `model fix`

#### 3.11 Cheat Sheet

| Command | Purpose | Common Example |
| --- | --- | --- |
| `./build/las init` | Initialize user config and baseline hardware info | `./build/las init --language zh-CN --api-key "sk-xxxx"` |
| `./build/las system init` | System init entry equivalent to `init` | `./build/las system init` |
| `./build/las system detect` | Trigger hardware detection | `./build/las system detect` |
| `./build/las system info` | Inspect system info | `./build/las system info` |
| `./build/las module list` | List manageable modules | `./build/las module list` |
| `./build/las module install <module>` | Install a module | `./build/las module install ollama` |
| `./build/las module update <module>` | Upgrade a module | `./build/las module update llama.cpp` |
| `./build/las module uninstall <module>` | Uninstall a module | `./build/las module uninstall vllm` |
| `./build/las module purge <module>` | Deep-clean a module | `./build/las module purge ollama` |
| `./build/las module check <module>` | Check module health | `./build/las module check comfyui` |
| `./build/las module setting <module> ...` | Run module-specific configuration | `./build/las module setting comfyui Comfy-Org_z_image_turbo` |
| `./build/las module config-plan <module>` | Generate a module config plan | `./build/las module config-plan vllm --planner-debug --dry-run` |
| `./build/las service start <service>` | Start a service | `./build/las service start ollama` |
| `./build/las service stop <service>` | Stop a service | `./build/las service stop ollama` |
| `./build/las service status <service>` | Show service status | `./build/las service status ollama` |
| `./build/las provider list` | List built-in LLM providers | `./build/las provider list` |
| `./build/las model search <query>` | Search models | `./build/las model search qwen3 --source huggingface --limit 20` |
| `./build/las model download <model-id>` | Download a model | `./build/las model download unsloth/Qwen3-Coder-Next-GGUF --file Q4_K_M.gguf` |
| `./build/las model list` | List downloaded models | `./build/las model list` |
| `./build/las model repair <model-id>` | Repair model support files | `./build/las model repair ByteDance/Ouro-2.6B-Thinking` |
| `./build/las model rm <model-id>` | Remove a model | `./build/las model rm qwen3-coder:30b --force` |
| `./build/las model run <model-id>` | Start a local model | `./build/las model run unsloth/Qwen3-Coder-Next-GGUF --ctx-size 65536 --threads 16` |
| `./build/las model run <model-id> --auto-batch` | Auto-tune batch / ubatch | `./build/las model run unsloth/Qwen3-Coder-Next-GGUF --auto-batch --dry-run` |
| `./build/las model run <model-id> --smart-run` | Use smart-run to suggest runtime parameters | `./build/las model run unsloth/Qwen3-Coder-Next-GGUF --smart-run --smart-run-debug` |
| `./build/las model run <model-id> --smart-run-refresh` | Ignore cache and ask the LLM again | `./build/las model run unsloth/Qwen3-Coder-Next-GGUF --smart-run --smart-run-refresh --dry-run` |
| `./build/las model smart-run-cache list` | List smart-run cache entries | `./build/las model smart-run-cache list unsloth/Qwen3-Coder-Next-GGUF` |
| `./build/las model smart-run-cache rm <model-id>` | Remove smart-run cache for a model | `./build/las model smart-run-cache rm unsloth/Qwen3-Coder-Next-GGUF` |
| `./build/las failure list` | List failure records | `./build/las failure list --phase smart_run --category timeout` |
| `./build/las failure show <event-id>` | Show failure details and suggestions | `./build/las failure show evt-xxxx` |

#### 3.12 Command Reference

##### Root Command

* `./build/las --help`: show full help
* Global flags:
  * `--config <path>`: specify the config file path
  * `--verbose`: enable verbose output

##### `init` / `system init`

Purpose:

* Initialize user configuration
* Write `~/.localaistack/config.yaml`
* Generate `~/.localaistack/base_info.json`

Common flags:

* `--config-path <path>`: specify where to write the config
* `--api-key <key>`: set the SiliconFlow API key
* `--language <lang>`: set the interaction language, such as `zh-CN` or `en`
* `--assistant-provider <name>` / `--assistant-model <id>`: set the assistant model
* `--assistant-base-url <url>` / `--assistant-timeout-seconds <n>`: set assistant access parameters
* `--translation-provider <name>` / `--translation-model <id>`: set the translation model
* `--translation-base-url <url>` / `--translation-timeout-seconds <n>`: set translation access parameters

##### `module`

Purpose:

* Manage software modules, including install, upgrade, uninstall, purge, health checks, and parameter settings

Subcommands:

* `module list`: list manageable modules and show whether they are installed
* `module install <module>`: install a module
* `module update <module>`: upgrade a module
* `module uninstall <module>`: uninstall a module
* `module purge <module>`: deep-clean a module
* `module check <module>`: validate whether a module is usable
* `module setting <module> <setting-args...>`: invoke module-specific setting logic
* `module config-plan <module>`: generate a configuration plan for a module

Common `module config-plan` flags:

* `--model <model-id>`: include a model ID in the planning context
* `--apply`: save to `~/.localaistack/config-plans/<module>.json`
* `--dry-run`: print only, do not save
* `--planner-debug`: print planning source and reasoning
* `--planner-strict`: fail immediately if planning fails
* `--output text|json`: set output format

##### `service`

Purpose:

* Unified service lifecycle entry point

Subcommands:

* `service start <service>`
* `service stop <service>`
* `service status <service>`

This is suitable for background services managed by LocalAIStack, such as `ollama`.

##### `model`

Purpose:

* Search, download, list, run, repair, and remove models

Subcommands:

* `model search <query>`
  * Flags: `--source, -s all|ollama|huggingface|modelscope`
  * Flags: `--limit, -n <N>`
* `model download <model-id> [file]`
  * Flags: `--source, -s <source>`
  * Flags: `--file, -f <filename>`
* `model list`
* `model rm <model-id>`
  * Flags: `--force, -f`
  * Flags: `--source, -s <source>`
* `model repair <model-id>`
  * Flags: `--source, -s <source>`
* `model run <model-id> [gguf-file-or-quant]`
  * Runtime routing: `GGUF -> llama.cpp`, `safetensors -> vLLM`

Common `model run` flags:

* Model source and file selection:
  * `--source, -s <source>`
  * `--file, -f <gguf-file>`
* llama.cpp inference parameters:
  * `--threads`
  * `--ctx-size`
  * `--n-gpu-layers`
  * `--tensor-split`
  * `--batch-size`
  * `--ubatch-size`
  * `--auto-batch`
* Sampling parameters:
  * `--temperature`
  * `--top-p`
  * `--top-k`
  * `--min-p`
  * `--presence-penalty`
  * `--repeat-penalty`
  * `--chat-template-kwargs`
* vLLM parameters:
  * `--vllm-max-model-len`
  * `--vllm-gpu-memory-utilization`
  * `--vllm-trust-remote-code`
* Common runtime parameters:
  * `--host`
  * `--port`
  * `--dry-run`

smart-run-related flags:

* `--smart-run`: enable hardware- and model-aware parameter suggestions
* `--smart-run-debug`: print parameter sources and fallback reasons
* `--smart-run-refresh`: ignore local cache and force a fresh LLM recommendation
* `--smart-run-strict`: fail immediately if smart-run fails

smart-run parameter persistence:

* Cache directory: `~/.localaistack/smart-run/`
* Save timing: immediately after the model process starts successfully
* Priority order:
  * Explicit user flags
  * Locally saved smart-run parameters
  * Fresh LLM suggestions
  * Static defaults / auto-tune

##### `model smart-run-cache`

Purpose:

* Inspect and remove persisted smart-run parameters

Subcommands:

* `model smart-run-cache list [model-id]`
  * Flags: `--runtime llama.cpp|vllm`
* `model smart-run-cache rm <model-id>`
  * Flags: `--runtime llama.cpp|vllm`

##### `provider`

Purpose:

* List built-in LLM providers

Subcommands:

* `provider list`

##### `failure`

Purpose:

* Inspect failure records and suggested actions

Subcommands:

* `failure list`
  * Flags: `--limit <N>`
  * Flags: `--phase <phase>`
  * Flags: `--category <category>`
  * Flags: `--output text|json`
* `failure show <event-id>`

Common `phase` values:

* `install_planner`
* `config_planner`
* `smart_run`
* `module_install`
* `model_run`

##### `system`

Purpose:

* System initialization and information entry points

Subcommands:

* `system init`: equivalent to root `init`
* `system detect`: hardware detection entry point
* `system info`: system information entry point

## Open Source

LocalAIStack is an open-source project.

* License: Apache License 2.0. See [LICENSE](./LICENSE).
* Contributions are welcome
* Vendor neutrality is a design principle

## Documentation

More documentation lives under `docs/`:

* [Architecture Design](./docs/architecture.md)
* [Module System and Manifest Specification](./docs/modules.md)
* [Hardware Capabilities and Policy Mapping](./docs/policies.md)
* [Runtime Execution Model](./docs/runtime.md)
* [Feature List](./docs/features.cn.md)

## Philosophy

LocalAIStack treats **local AI computing as infrastructure**, not as a loose collection of tools.

It aims to make local AI systems:

* Predictable
* Maintainable
* Understandable
* Sustainable for long-term use
