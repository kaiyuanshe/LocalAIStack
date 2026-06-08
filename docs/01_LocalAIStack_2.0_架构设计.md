# LocalAIStack 2.0 架构设计

版本：0.1 Draft  
日期：2026-04-28  
定位：面向个人与小团队的本地 AI 推理、RAG 与 Search 应用栈。

---

## 1. 背景与设计目标

LocalAIStack 2.0 的核心判断是：本地 LLM 推理不是单点框架选择，而是“硬件 × 驱动/runtime × inference engine × 模型格式 × attention/quant/KV 策略 × serving 配置”的组合问题。

但第一阶段不能把所有组合平铺展开，否则工程范围会失控。因此 2.0 采用两级抽象：

1. **Inference Recipe**：解决“LLM 如何在指定硬件上稳定跑起来”。
2. **Application Recipe**：解决“LLM 如何被用于 RAG、Search、Agent、知识库问答等应用”。

第一阶段只以三个整合型 inference engine 为核心：

- **vLLM**：面向 GPU serving、高吞吐、OpenAI-compatible API。
- **SGLang**：面向低延迟、高吞吐、prefix/cache-heavy、结构化生成与 Agent workload。
- **llama.cpp**：面向本地低依赖、GGUF、CPU/Apple/低显存与跨硬件推理。

其他组件，例如 CUDA、ROCm、Metal、Triton、FlashAttention、FlashInfer、AWQ、GPTQ、GGUF、FP8、KV cache、continuous batching、RadixAttention 等，均作为 recipe 的组成维度，而不是第一阶段的独立产品对象。

---

## 2. 总体架构

```text
LocalAIStack 2.0
├── Product Tier
│   ├── Express：少量 verified recipes，面向一般用户
│   └── Expert：动态 recipe compiler，面向高级用户
│
├── Recipe Layer
│   ├── Inference Recipe：LLM 如何跑
│   └── Application Recipe：RAG/Search/Agent 如何跑
│
├── Runtime Layer
│   ├── Dockerized Runtime：Linux/NVIDIA/CPU 优先
│   ├── Native Runtime：Apple Silicon / Windows fallback
│   └── Compose Package：多服务应用编排
│
├── LLM Inference Core
│   ├── vLLM
│   ├── SGLang
│   └── llama.cpp
│
├── Model Services
│   ├── Generation Service
│   ├── Embedding Service
│   ├── Reranker Service
│   ├── Classification Service
│   └── Tool/Function Calling Service
│
├── Knowledge & Search Layer
│   ├── Source Connector
│   ├── Parser / OCR / Converter
│   ├── Chunker
│   ├── Metadata Extractor
│   ├── Vector Index
│   ├── Keyword Index
│   ├── Hybrid Retriever
│   ├── Reranker
│   └── Citation / Source Tracing
│
├── Orchestration Layer
│   ├── RAG Gateway
│   ├── Search Gateway
│   ├── Query Rewrite / Expansion
│   ├── Context Packing
│   ├── Agent Workflow
│   └── Guardrails
│
└── Ops & Evaluation Layer
    ├── Hardware Detector
    ├── Preflight Validator
    ├── Healthcheck
    ├── Benchmark Runner
    ├── Diagnostics
    ├── Observability
    └── Recipe Registry
```

---

## 3. 核心设计原则

### 3.1 Docker-first，但不是 Docker-only

LocalAIStack 2.0 第一阶段采用 Docker-first 的 recipe 载体，尤其面向 Linux + NVIDIA 场景。Docker 用来隔离 Python、PyTorch、CUDA、vLLM、SGLang、FlashAttention、FlashInfer 等依赖冲突。

但 Docker 不是所有场景的最佳解：

| 场景 | 推荐载体 |
|---|---|
| Linux + NVIDIA GPU | Docker-first |
| Linux + CPU | Docker-first |
| AMD ROCm | Docker-supported，第二阶段强化 |
| Apple Silicon | Native-first，优先 Metal / llama.cpp / Ollama / brew |
| Windows | WSL2 + Docker 或 native fallback |

### 3.2 自研 recipe management，不自研 inference engine

LocalAIStack 不重写 vLLM、SGLang、llama.cpp，也不手搓 attention kernel。自研部分是：

- Recipe Registry
- Recipe Selector
- Hardware Detector
- Preflight Validator
- Healthcheck
- Benchmark Runner
- Diagnostics
- Expert Plugin Interface
- Application Recipe Compiler

### 3.3 Inference engine 与 RAG/Search 解耦

vLLM/SGLang/llama.cpp 只负责 generation。RAG/Search 不应塞进底层 engine，而应通过 RAG Gateway 或 Search Gateway 调用 generation service。

```text
User Query
→ RAG/Search Gateway
→ Retrieval / Rerank / Context Packing
→ Generation Service
→ Answer with citations
```

### 3.4 Embedding、Reranker、Generation 是独立服务

RAG/Search 系统不应假设 embedding、reranker、generation 使用同一个模型或同一个 runtime。

```text
Generation Model: vLLM / SGLang / llama.cpp
Embedding Model: TEI / sentence-transformers / ONNX / local service
Reranker Model: cross-encoder / BGE reranker / Jina reranker / LLM reranker
```

### 3.5 Express sells certainty, Expert sells control

Express 的目标是稳定、少选择、低失败率。Expert 的目标是可组合、可解释、可调优。

---

## 4. Product Tier 架构

## 4.1 Express

Express 是 verified Docker/native recipe catalog。

目标体验：

```text
detect hardware
→ recommend recipe
→ docker compose up / native run
→ healthcheck
→ OpenAI-compatible API ready
```

Express 只暴露少量安全选项：

- 模型档位：Small / Medium / Large
- 用途：Chat / API / RAG
- 内存策略：safe / balanced / aggressive
- 上下文长度：conservative / balanced / large

Express 不暴露：

- FlashAttention vs FlashInfer
- max_num_batched_tokens
- max_num_seqs
- kv_cache_dtype
- CUDA graphs
- speculative decoding
- tensor parallel
- specific kernel backend

## 4.2 Expert

Expert 是动态 recipe compiler。

输入：

```yaml
hardware:
  auto_detect: true
model:
  id: Qwen/Qwen3-32B-AWQ
workload:
  type: openai_api
  context_length: 32768
  concurrency: 8
  priority: latency_throughput_balance
constraints:
  max_vram_gb: 24
  local_only: true
```

输出：

```yaml
candidate_recipes:
  - engine: vllm
    confidence: high
    reason: NVIDIA GPU + HF safetensors + API serving
  - engine: sglang
    confidence: medium
    reason: useful if prefix-heavy or structured generation
fallbacks:
  - reduce max_model_len to 16384
  - reduce max_num_seqs
  - switch to smaller model
artifacts:
  - docker-compose.yaml
  - recipe.yaml
  - healthcheck.sh
  - benchmark.sh
```

---

## 5. Recipe 模型

## 5.1 Inference Recipe

Inference Recipe 描述某个 LLM 在某类硬件上的运行方式。

```yaml
id: express.nvidia.rtx4090.qwen32b.vllm.awq
tier: express
kind: inference
engine: vllm
status: verified

target:
  hardware:
    vendor: nvidia
    min_vram_gb: 24
    recommended_gpu:
      - rtx_4090
      - rtx_3090
      - rtx_5090
  os:
    - linux
  runtime:
    docker: true
    cuda: "12.x"

model:
  source: huggingface
  id: Qwen/Qwen3-32B-AWQ
  family: qwen
  params: 32b
  format: safetensors
  quantization: awq

runtime:
  image: localaistack/vllm:cuda12
  api: openai-compatible
  port: 8000

engine_config:
  max_model_len: 32768
  gpu_memory_utilization: 0.90
  tensor_parallel_size: 1
  enable_prefix_caching: true

validation:
  healthcheck: ./healthcheck.sh
  benchmark: ./benchmark.sh

fallbacks:
  oom:
    - reduce max_model_len to 16384
    - reduce max_num_seqs
    - switch to qwen14b
```

## 5.2 Application Recipe

Application Recipe 组合 inference、embedding、index、retrieval、rerank、context packing、UI/API。

```yaml
id: express.rag.local-docs.qwen14b.qdrant
kind: application
tier: express
status: verified

llm:
  recipe: express.nvidia.rtx4090.qwen14b.vllm

embedding:
  provider: local
  engine: text-embeddings-inference
  model: BAAI/bge-m3

knowledge:
  sources:
    - local_folder
  parsers:
    - markdown
    - txt
    - pdf
  chunking:
    strategy: recursive_or_heading
    chunk_size: 800
    overlap: 120

retrieval:
  vector_store: qdrant
  keyword_search: optional
  hybrid_search: enabled
  top_k: 20
  rerank_top_k: 5

answering:
  citations: required
  fallback_when_no_context: say_not_found
  allow_freeform_answer: false

services:
  rag_api: true
  web_ui: open-webui
```

---

## 6. Engine 适配层

## 6.1 vLLM Adapter

适用场景：

- NVIDIA GPU / 高吞吐 API serving
- Hugging Face safetensors 模型
- 多请求并发
- OpenAI-compatible API
- batch inference / RAG backend / Agent backend

核心 recipe 维度：

- CUDA/ROCm/XPU/CPU runtime
- PyTorch version
- attention backend
- quant backend
- KV cache dtype
- max_model_len
- max_num_seqs
- max_num_batched_tokens
- tensor parallel / pipeline parallel

## 6.2 SGLang Adapter

适用场景：

- Agent workload
- structured output
- prefix/cache-heavy workload
- 多轮共享系统 prompt
- DeepSeek/Qwen/Llama 等模型上的低延迟 serving

核心 recipe 维度：

- RadixAttention / prefix cache
- attention backend
- speculative decoding
- structured output
- MoE / MLA / VLM 支持
- tensor/pipeline/expert parallel

## 6.3 llama.cpp Adapter

适用场景：

- GGUF
- CPU-only
- Apple Silicon / Metal
- 低显存 GPU
- 本地聊天
- 简单 OpenAI-compatible server

核心 recipe 维度：

- backend：CPU / Metal / CUDA / HIP / Vulkan / SYCL / OpenVINO
- GGUF quant：Q4_K_M / Q5_K_M / Q6 / Q8 / IQ
- n_gpu_layers
- ctx_size
- threads
- batch size
- mmap / mlock

---

## 7. RAG/Search 扩展架构

## 7.1 Index-time Pipeline

```text
source data
→ connector
→ parser / OCR / converter
→ cleaner
→ chunker
→ metadata extractor
→ embedding service
→ vector index
→ keyword index
→ document store
```

## 7.2 Query-time Pipeline

```text
user query
→ query normalization
→ query rewrite / expansion
→ router
→ retrieval
→ metadata filtering
→ reranking
→ context packing
→ prompt construction
→ LLM generation
→ citation / verification
→ answer
```

## 7.3 插件接口

后续 Expert 需要这些插件接口：

```text
source_connector plugin
parser plugin
chunker plugin
embedding plugin
index plugin
retriever plugin
reranker plugin
context_packer plugin
generator plugin
evaluation plugin
diagnostics plugin
```

统一文档块接口：

```python
@dataclass
class DocumentChunk:
    id: str
    text: str
    source_uri: str
    title: str | None
    metadata: dict
    score: float | None = None
```

统一 retriever 接口：

```python
class Retriever:
    def retrieve(self, query: str, filters: dict | None, top_k: int) -> list[DocumentChunk]:
        ...
```

---

## 8. Docker/Compose Artifact 结构

建议每个 recipe 目录包含：

```text
recipe.yaml
docker-compose.yaml
.env.example
README.md
healthcheck.sh
benchmark.sh
model-config.yaml
```

建议 repo 结构：

```text
localaistack/
  recipes/
    express/
      inference/
        nvidia/
        cpu/
        apple-native/
      rag/
        local-docs-qdrant/
        codebase-search/
    expert/
      templates/
      plugins/
  engines/
    vllm/
    sglang/
    llama_cpp/
  runtime/
    docker/
    native/
  registry/
  benchmarks/
  diagnostics/
  docs/
```

---

## 9. 验证、诊断与评测

## 9.1 Preflight Validator

启动前检查：

- Docker 是否安装
- NVIDIA Container Toolkit 是否可用
- GPU 是否可被容器看到
- CUDA driver 是否满足要求
- 模型文件 / Hugging Face token 是否可用
- 端口是否冲突
- 磁盘空间是否足够
- 显存是否可能 OOM

## 9.2 Healthcheck

启动后检查：

- 服务端口是否打开
- `/v1/models` 是否返回
- `/v1/chat/completions` 是否能生成
- 首 token 是否超时
- 是否返回可解析 JSON

## 9.3 Benchmark

基础指标：

- TTFT
- prefill tokens/s
- decode tokens/s
- peak VRAM
- peak RAM
- max stable context
- max stable concurrency
- error rate

## 9.4 Diagnostics

错误分类：

- container cannot see GPU
- CUDA driver too old
- model download failed
- port conflict
- OOM at load time
- OOM at long context
- unsupported quantization
- unsupported model architecture
- tokenizer mismatch
- OpenAI API incompatible response

---

## 10. 分阶段落地

## Phase 1：LLM Express

目标：10-20 个 verified inference recipes。

优先 recipe：

- RTX 4090 + Qwen 14B/32B + vLLM
- RTX 3060 12GB + GGUF Q4/Q5 + llama.cpp CUDA
- CPU-only + Llama/Qwen 7B/8B Q4 + llama.cpp
- Apple M-series + GGUF + llama.cpp native
- RTX 4090 + SGLang + Qwen/DeepSeek distill

## Phase 2：RAG Express

目标：3 个应用套餐。

- Local Docs RAG
- Personal Knowledge Base
- Codebase Search

默认组合：

```text
LLM recipe
+ embedding service
+ Qdrant
+ RAG API
+ Open WebUI or simple UI
```

## Phase 3：Expert Rule-based Compiler

目标：根据 hardware facts、model facts、workload facts 自动生成 candidate recipes。

## Phase 4：Expert Plugin System

目标：开放 connector/parser/chunker/embedding/index/retriever/reranker/generator/eval 插件。

## Phase 5：Benchmark-driven Optimizer

目标：基于真实 benchmark 结果自动调参和推荐 fallback。

---

## 11. 架构结论

LocalAIStack 2.0 的核心不是“再写一个推理框架”，而是围绕 vLLM、SGLang、llama.cpp 构建可验证、可复现、可选择、可诊断、可扩展的 recipe management system。

第一阶段以 Express 的 Docker/native verified recipes 解决成功率；第二阶段通过 Expert 的插件体系和 recipe compiler 解决覆盖率与调优能力。
