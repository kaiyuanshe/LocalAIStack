# LocalAIStack 2.0 产品说明书

版本：0.1 Draft  
日期：2026-04-28  
产品定位：让个人和小团队用最小运维成本，在本地硬件上运行 LLM、RAG、Search 与 Agent 应用。

---

## 1. 产品一句话

LocalAIStack 2.0 是一个本地 AI 配方系统：

```text
Express：给一般用户的一键本地 AI 套餐。
Expert：给高级用户的动态推理/RAG/Search 配方生成器。
```

它不重新实现 LLM 推理引擎，而是将 vLLM、SGLang、llama.cpp 以及 RAG/Search 相关组件组织成可运行、可验证、可 benchmark、可复用的 recipes。

---

## 2. 用户问题

普通用户的问题：

- 我有一台电脑，能不能本地跑大模型？
- 我该用 vLLM、SGLang、llama.cpp 还是 Ollama？
- 模型多大合适？会不会爆显存？
- 怎么用 Docker 跑起来？
- 怎么接 OpenAI-compatible API？
- 怎么做本地文档问答？

高级用户的问题：

- RTX 4090 上 Qwen 32B 应该用什么 quant 和 serving engine？
- 32K context + 8 并发怎么避免 OOM？
- RAG 应该选什么 embedding、vector DB、reranker？
- Agent workload 是否该用 SGLang 的 prefix cache？
- 怎么比较 vLLM 和 SGLang 在同一模型上的效果？
- 如何把某个硬件/模型/工作负载固化成可复现 recipe？

---

## 3. 产品目标

## 3.1 Express 目标

Express 面向一般用户，核心承诺是：

```text
少量选择，稳定跑起来，默认不踩坑。
```

Express 不追求覆盖所有模型和硬件，也不追求极致性能。它追求：

- 首次启动成功率
- 配置简单
- 依赖隔离
- 明确的 fallback
- 可直接调用的 API
- 可验证的健康检查

## 3.2 Expert 目标

Expert 面向高级用户，核心承诺是：

```text
用户点菜，系统生成可解释、可运行、可调优的 recipe。
```

Expert 追求：

- 动态配方生成
- 多 engine 比较
- 软硬件兼容性推断
- 量化与上下文策略推荐
- RAG/Search 插件组合
- benchmark-driven tuning

---

## 4. 产品分层

## 4.1 LocalAIStack Express

### 目标用户

- 不想理解 CUDA/PyTorch/vLLM 依赖的人
- 希望本地跑 LLM API 的开发者
- 希望本地文档问答的个人用户
- 小团队内部 AI 工具部署者

### 典型体验

```bash
localaistack express detect
localaistack express recommend
localaistack express run qwen-14b
localaistack express healthcheck
localaistack express benchmark
```

### 用户只需要选择

- 硬件类型：NVIDIA / CPU / Apple / AMD later
- 模型档位：Small / Medium / Large
- 用途：Chat / API / RAG / Search
- 内存策略：Safe / Balanced / Aggressive

### 系统自动决定

- engine：vLLM / SGLang / llama.cpp
- container image / native runtime
- model format
- quant choice
- context length
- default runtime flags
- fallback 策略

## 4.2 LocalAIStack Expert

### 目标用户

- 高级本地 AI 用户
- AI infra 开发者
- RAG/Search 系统开发者
- Agent 框架开发者
- 小团队平台工程师

### 典型体验

```bash
localaistack expert plan \
  --hardware auto \
  --model Qwen/Qwen3-32B-AWQ \
  --workload openai-api \
  --context 32768 \
  --concurrency 8

localaistack expert build recipe.yaml
localaistack expert run recipe.yaml
localaistack expert tune recipe.yaml
```

### Expert 输出

- 候选 recipes
- 推荐理由
- 风险说明
- Docker Compose
- native runner，若适用
- healthcheck
- benchmark plan
- fallback plan

---

## 5. 核心产品模块

## 5.1 Recipe Registry

存储所有 verified / experimental / deprecated recipes。

状态定义：

| 状态 | 含义 |
|---|---|
| verified | 已在指定硬件/模型/版本上验证 |
| experimental | 可以尝试，但成功率或性能未充分验证 |
| deprecated | 旧版本，保留兼容 |
| broken | 已知不可用 |

## 5.2 Hardware Detector

检测内容：

- OS
- CPU 架构与指令集
- RAM
- GPU vendor/model/VRAM
- NVIDIA driver
- CUDA availability
- Docker availability
- NVIDIA Container Toolkit availability
- Apple Metal availability
- ROCm availability，后续

## 5.3 Recipe Selector

根据用户机器和目标用途推荐 recipe。

输入：

```yaml
hardware:
  gpu: RTX 4090
  vram_gb: 24
  docker_gpu_available: true
intent:
  workload: openai_api
  model_size: 14b
  memory_mode: balanced
```

输出：

```yaml
recommendations:
  - id: express.nvidia.rtx4090.qwen14b.vllm
    confidence: high
    reason: fits 24GB VRAM and API serving
  - id: express.nvidia.rtx4090.qwen32b.awq.vllm
    confidence: medium
    reason: possible but memory-sensitive
```

## 5.4 Runtime Runner

负责运行 recipe。

运行模式：

- Docker Compose
- Docker single container
- Native binary
- WSL2 fallback

## 5.5 Healthcheck

检查 recipe 是否真正可用，而不是“容器启动了就算成功”。

必须检查：

- API port ready
- model loaded
- `/v1/models` works
- `/v1/chat/completions` works
- generation latency within threshold

## 5.6 Benchmark

用于记录真实机器上的 recipe 表现。

基础指标：

- TTFT
- decode tokens/s
- prefill tokens/s
- peak VRAM
- peak RAM
- max stable concurrency
- max stable context length

## 5.7 Diagnostics

将错误日志翻译成人能理解的原因和操作。

示例：

```text
检测到容器无法访问 GPU。
可能原因：未安装 NVIDIA Container Toolkit，或 Docker daemon 未配置 nvidia runtime。
建议：安装 toolkit 后运行 nvidia-ctk runtime configure --runtime=docker。
```

---

## 6. Express 首批产品包

## 6.1 LLM API Express

面向：想本地起一个 OpenAI-compatible API 的用户。

默认组合：

```text
vLLM or SGLang or llama.cpp server
+ Docker/native runner
+ healthcheck
+ benchmark
```

首批 recipe：

| Recipe | 硬件 | Engine | 模型类型 |
|---|---|---|---|
| qwen14b-vllm-nvidia24g | RTX 3090/4090/5090 | vLLM | HF safetensors |
| qwen32b-awq-vllm-nvidia24g | RTX 4090/5090 | vLLM | AWQ |
| llama8b-llamacpp-cpu-q4 | CPU-only | llama.cpp | GGUF Q4 |
| qwen14b-llamacpp-cuda-q4 | RTX 3060 12GB+ | llama.cpp | GGUF Q4/Q5 |
| qwen14b-llamacpp-metal-q4 | Apple M-series | llama.cpp native | GGUF Q4/Q5 |

## 6.2 Local Docs RAG Express

面向：想对本地文档提问的用户。

默认组合：

```text
LLM API recipe
+ embedding service
+ Qdrant
+ parser
+ chunker
+ RAG API
+ simple UI / Open WebUI
```

支持数据源：

- Markdown
- TXT
- PDF
- local folder

默认行为：

- 自动索引指定目录
- 回答必须带引用
- 检索不到时明确说没有找到依据
- 默认不允许模型脱离文档自由编造

## 6.3 Local Search Express

面向：想做个人知识库搜索、代码库搜索的用户。

默认组合：

```text
embedding service
+ vector DB
+ optional keyword index
+ search API
+ search UI
```

不强制生成答案。

---

## 7. Expert 产品能力

## 7.1 Dynamic Inference Recipe Compiler

根据硬件、模型、workload 自动生成 inference recipe。

示例输入：

```text
RTX 4090, Qwen 32B AWQ, 32K context, 8 concurrency, OpenAI API
```

示例输出：

```text
推荐 vLLM，备选 SGLang。
风险：24GB 显存下 32K context + 8 并发可能 OOM。
默认启用 conservative max_num_seqs。
fallback：降低 context 至 16K 或改用 14B。
```

## 7.2 Dynamic Application Recipe Compiler

根据应用目标生成 RAG/Search/Agent recipe。

示例输入：

```text
本地 PDF + Markdown 文档问答，中文优先，答案带引用，RTX 4090。
```

示例输出：

```text
LLM: qwen14b-vllm-nvidia24g
Embedding: bge-m3 via TEI
Vector DB: Qdrant
Parser: Docling / PyMuPDF fallback
Chunker: heading-aware + recursive fallback
Reranker: optional bge-reranker
UI: Open WebUI
```

## 7.3 Plugin System

Expert 的插件层：

| 插件 | 作用 |
|---|---|
| hardware_detector | 识别硬件与 runtime |
| model_metadata | 识别模型架构、格式、量化、上下文 |
| engine_adapter | 生成 vLLM/SGLang/llama.cpp 启动方案 |
| container_builder | 构建或选择 Docker image |
| source_connector | 接入文件、GitHub、Notion、Drive 等 |
| parser | 文档解析 |
| chunker | 文档切块 |
| embedding | 向量化 |
| index | 存储与检索索引 |
| retriever | 检索策略 |
| reranker | 重排 |
| context_packer | 上下文压缩与排列 |
| evaluator | RAG/LLM 评测 |
| diagnostics | 错误解释与 fallback |

---

## 8. 非目标

LocalAIStack 2.0 第一阶段不做：

- 自研 LLM engine
- 自研 CUDA kernel
- 自研 vector database
- 自研完整 UI 平台
- 覆盖所有硬件
- 覆盖所有模型格式
- 自动量化全流程
- 企业级权限与多租户，除非进入后续商业版设计

---

## 9. 成功指标

## 9.1 Express 指标

- 首次启动成功率
- 15 分钟内完成本地 API 启动的比例
- healthcheck 通过率
- 用户遇到 OOM 后成功 fallback 的比例
- 文档理解成本
- verified recipe 数量

## 9.2 Expert 指标

- 动态生成 recipe 的可运行率
- 推荐 engine 的命中率
- benchmark-driven tuning 后性能提升
- plugin 覆盖度
- 用户手动改配置次数下降

## 9.3 RAG/Search 指标

- 检索命中率
- rerank 后相关性提升
- citation 准确率
- answer faithfulness
- no-context fallback 准确率
- indexing 成功率

---

## 10. 路线图

## Milestone 1：Express Inference MVP

产物：

- recipe registry
- hardware detector
- 5-10 个 verified recipes
- Docker Compose runner
- healthcheck
- simple benchmark

## Milestone 2：Express RAG MVP

产物：

- local-docs-RAG compose
- embedding service
- Qdrant
- parser/chunker
- RAG API
- citation output

## Milestone 3：Recipe Selector

产物：

- 硬件自动匹配 recipe
- fallback 推荐
- config generation

## Milestone 4：Expert Alpha

产物：

- rule-based recipe compiler
- engine adapters
- benchmark result store
- plugin interface draft

## Milestone 5：Expert Beta

产物：

- RAG/Search plugin ecosystem
- benchmark-driven optimizer
- community recipe submission
- compatibility matrix

---

## 11. 产品结论

LocalAIStack 2.0 的产品核心是 recipe，而不是 framework。

Express 提供少量可靠路线，让普通用户跑起来；Expert 提供可组合、可解释、可调优的动态配方系统，让高级用户针对不同模型、硬件、workload 构建最合适的本地 AI 应用。
