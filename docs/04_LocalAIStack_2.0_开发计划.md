# LocalAIStack 2.0 开发计划

版本：0.1 Draft  
日期：2026-04-28  
依据：

- `01_LocalAIStack_2.0_架构设计.md`
- `02_LocalAIStack_2.0_产品说明书.md`
- `03_后续开发项目列表.md`

---

## 1. 计划目标

LocalAIStack 2.0 的开发主线是围绕 recipe management system 建立一套可运行、可验证、可诊断、可 benchmark、可扩展的本地 AI 应用栈。

第一阶段不自研 LLM engine、CUDA kernel、vector database 或完整 UI 平台，而是把 vLLM、SGLang、llama.cpp、Docker Compose、Qdrant、embedding service 和可选 UI 组织成少量高可靠 recipes。

产品推进顺序：

```text
Express Inference
→ Express RAG/Search
→ Recipe Selector
→ Expert Rule-based Compiler
→ Expert Plugin System
→ Benchmark-driven Optimizer
```

---

## 2. 开发原则

1. **Express 优先成功率**
   - 首批只做少量 verified recipes。
   - 不暴露底层复杂参数。
   - 每个 recipe 必须有 preflight、healthcheck、benchmark 和 fallback。

2. **Expert 延后覆盖率**
   - 等 Express 的 recipe schema、runner、diagnostics 稳定后，再做动态生成。
   - Expert 初期采用 rule-based compiler，不引入过早的复杂优化器。

3. **Inference 与 RAG/Search 解耦**
   - vLLM、SGLang、llama.cpp 只作为 generation service。
   - embedding、reranker、retrieval、context packing 作为独立服务或模块。

4. **Docker-first，不是 Docker-only**
   - Linux + NVIDIA 和 Linux + CPU 走 Docker-first。
   - Apple Silicon 走 native-first llama.cpp/Metal。
   - Windows 先作为 WSL2/Docker 或 native fallback，不作为首批主线。

5. **先本地可复现，再扩大生态**
   - P0 项目用于第一阶段核心集成。
   - P1 项目用于 RAG/Search。
   - P2 项目用于 Expert 插件和长期优化。

---

## 3. 总体里程碑

| 阶段 | 目标 | 核心产物 | 建议优先级 |
|---|---|---|---|
| M0：工程地基 | 建立 recipe、CLI、运行与测试骨架 | repo 结构、schema、CLI skeleton、测试框架 | P0 |
| M1：Express Inference MVP | 本地 LLM API 稳定跑起来 | 5-10 个 verified inference recipes、runner、healthcheck、benchmark | P0 |
| M2：Express RAG MVP | 本地文档问答可运行 | local-docs-RAG compose、embedding、Qdrant、parser/chunker、RAG API、citation | P0/P1 |
| M3：Recipe Selector | 根据硬件和用途自动推荐 recipe | hardware detector、selector、fallback、config generation | P0 |
| M4：Expert Alpha | 生成可解释的候选 recipes | rule-based compiler、engine adapters、benchmark result store | P1 |
| M5：Expert Beta | 插件化与调优闭环 | plugin interface、RAG/Search plugins、benchmark-driven optimizer、community recipe flow | P1/P2 |

---

## 4. M0：工程地基

### 4.1 目标

建立后续所有阶段共用的工程骨架，先把 recipe 作为一等对象固化下来。

### 4.2 工作包

| 工作包 | 内容 | 交付物 |
|---|---|---|
| Repo 结构整理 | 按架构设计建立 recipes、engines、runtime、registry、benchmarks、diagnostics 目录 | 基础目录与 README |
| Recipe schema | 定义 inference recipe 与 application recipe 的 YAML schema | `recipe.schema.yaml`、示例 recipe |
| CLI skeleton | 定义 `localaistack express/expert` 命令结构 | CLI 基础命令、帮助文档 |
| Config loader | 加载、校验、解析 recipe | schema validation、错误输出 |
| 测试框架 | 为 schema、selector、runner 预留测试入口 | 单元测试和 smoke test |

### 4.3 验收标准

- 可以校验一个 inference recipe YAML。
- 可以校验一个 application recipe YAML。
- CLI 能列出 registry 中的 recipes。
- recipe 校验失败时能输出明确字段错误。

---

## 5. M1：Express Inference MVP

### 5.1 目标

让用户在首批硬件上用最少步骤启动 OpenAI-compatible API，并确认模型真实可用。

### 5.2 首批 recipes

| Recipe | 硬件 | Engine | Runtime | 模型格式 |
|---|---|---|---|---|
| `qwen14b-vllm-nvidia24g` | RTX 3090/4090/5090 | vLLM | Docker | HF safetensors |
| `qwen32b-awq-vllm-nvidia24g` | RTX 4090/5090 | vLLM | Docker | AWQ |
| `llama8b-llamacpp-cpu-q4` | CPU-only | llama.cpp | Docker/native | GGUF Q4 |
| `qwen14b-llamacpp-cuda-q4` | RTX 3060 12GB+ | llama.cpp | Docker | GGUF Q4/Q5 |
| `qwen14b-llamacpp-metal-q4` | Apple M-series | llama.cpp | native | GGUF Q4/Q5 |
| `qwen-or-deepseek-sglang-nvidia24g` | RTX 4090/5090 | SGLang | Docker | HF safetensors/quant |

### 5.3 工作包

| 工作包 | 内容 | 交付物 |
|---|---|---|
| vLLM adapter | 生成 vLLM 启动参数与 compose | vLLM recipe template |
| SGLang adapter | 生成 SGLang 启动参数与 compose | SGLang recipe template |
| llama.cpp adapter | 支持 CPU、CUDA、Metal 参数 | llama.cpp recipe template |
| Docker runner | 启动、停止、查看日志 | `express run/stop/logs` |
| Native runner | Apple Metal 和 CPU fallback | native run script |
| Preflight validator | 检测 Docker、GPU、driver、端口、磁盘、显存风险 | `express preflight` |
| Healthcheck | 检查 `/v1/models`、`/v1/chat/completions`、JSON、首 token 超时 | `express healthcheck` |
| Simple benchmark | TTFT、decode tps、prefill tps、peak VRAM/RAM、错误率 | `express benchmark` |
| Diagnostics | 识别 GPU 不可见、driver 过旧、OOM、模型下载失败、端口冲突 | diagnostics rules |

### 5.4 验收标准

- 至少 5 个 inference recipes 达到 verified 状态。
- Linux + NVIDIA recipe 能通过 preflight、run、healthcheck、benchmark。
- CPU 或 Apple recipe 至少有一条非 GPU fallback 路线。
- healthcheck 不只检查容器存活，必须完成一次真实 chat completion。
- OOM、端口冲突、Docker GPU 不可见等常见错误能给出明确处理建议。

---

## 6. M2：Express RAG MVP

### 6.1 目标

提供本地文档问答套餐，让用户能索引 Markdown、TXT、PDF 和本地文件夹，并得到带引用的回答。

### 6.2 默认技术组合

```text
LLM API recipe
+ embedding service
+ Qdrant
+ parser/chunker
+ RAG API
+ Open WebUI or simple UI
```

首版建议：

- Vector DB：Qdrant。
- Embedding：优先 TEI 或 sentence-transformers。
- Parser：Markdown/TXT 内置，PDF 先用 PyMuPDF，Docling 作为增强候选。
- UI：API-first，Open WebUI 作为可选组件。

### 6.3 工作包

| 工作包 | 内容 | 交付物 |
|---|---|---|
| Application recipe schema | 描述 LLM、embedding、knowledge、retrieval、answering、services | application recipe schema |
| Local Docs RAG compose | 编排 LLM、embedding、Qdrant、RAG API | compose package |
| Ingestion pipeline | source connector、parser、chunker、metadata extractor、embedding、index | indexing command |
| Query pipeline | query normalization、retrieval、rerank optional、context packing、prompt construction | RAG API |
| Citation | 输出 source URI、title、chunk id、引用片段 | citation response format |
| No-context fallback | 无上下文时明确回答找不到依据 | answering policy |
| RAG healthcheck | 检查 indexing、retrieval、generation、citation | RAG smoke test |

### 6.4 验收标准

- 能索引本地目录中的 Markdown、TXT、PDF。
- 查询结果必须包含 citation。
- 检索不到时不能自由编造。
- RAG API 与 generation service 解耦。
- embedding service 与 generation engine 不绑定。

---

## 7. M3：Recipe Selector

### 7.1 目标

把 Express 从“用户手动选 recipe”推进到“系统根据硬件和用途推荐 recipe”。

### 7.2 工作包

| 工作包 | 内容 | 交付物 |
|---|---|---|
| Hardware detector | 检测 OS、CPU、RAM、GPU、VRAM、driver、CUDA、Docker、NVIDIA Container Toolkit、Metal | hardware facts JSON |
| Intent model | 表达 workload、model size、memory mode、context profile | intent schema |
| Selection rules | 根据 hardware facts + intent 匹配 recipe | selector rules |
| Confidence scoring | high/medium/low 推荐等级和 reason | recommendation output |
| Fallback planner | OOM、driver、端口、模型不兼容时给备选方案 | fallback list |
| Config generator | 生成 `.env`、compose override、runtime args | config artifacts |

### 7.3 验收标准

- `express detect` 能输出稳定的 hardware facts。
- `express recommend` 能给出 recipe、confidence、reason、fallback。
- 推荐结果能区分 NVIDIA Docker、CPU-only、Apple native。
- 24GB VRAM + 32B + 32K context + 高并发场景能标注 OOM 风险。

---

## 8. M4：Expert Alpha

### 8.1 目标

实现 rule-based dynamic recipe compiler，让高级用户输入硬件、模型、workload、约束后得到可解释候选 recipe。

### 8.2 工作包

| 工作包 | 内容 | 交付物 |
|---|---|---|
| Expert input schema | hardware、model、workload、constraints | expert plan schema |
| Model metadata resolver | 识别模型 family、params、format、quantization、context | model facts |
| Engine adapters | vLLM、SGLang、llama.cpp 启动方案生成 | adapter output |
| Candidate generation | 生成 engine 候选、参数候选、fallback 候选 | candidate recipes |
| Explanation engine | 输出推荐理由、风险说明、限制条件 | plan report |
| Build artifacts | 输出 compose、recipe、healthcheck、benchmark | `expert build` |
| Benchmark store | 记录 benchmark 结果，为后续 optimizer 做准备 | local result store |

### 8.3 验收标准

- `expert plan` 能为指定硬件、模型、workload 生成至少 2 个候选方案。
- 每个候选方案必须包含推荐理由和风险。
- `expert build` 能把候选方案落成可运行 recipe artifacts。
- benchmark 结果能关联 hardware facts、model facts、recipe id 和 runtime config。

---

## 9. M5：Expert Beta

### 9.1 目标

建立插件体系和 benchmark-driven tuning 闭环，提升覆盖率、可扩展性和高级调优能力。

### 9.2 插件范围

首批插件接口：

- `source_connector`
- `parser`
- `chunker`
- `embedding`
- `index`
- `retriever`
- `reranker`
- `context_packer`
- `generator`
- `evaluation`
- `diagnostics`

### 9.3 工作包

| 工作包 | 内容 | 交付物 |
|---|---|---|
| Plugin interface draft | 定义插件生命周期、输入输出、配置、错误模型 | plugin spec |
| RAG/Search plugins | Qdrant、TEI/sentence-transformers、PyMuPDF/Docling、reranker optional | plugin implementations |
| Evaluation integration | 引入轻量 RAG eval，Ragas/DeepEval 后置 | eval runner |
| Benchmark optimizer | 基于真实结果调整 context、concurrency、batch、memory mode | tuning suggestions |
| Compatibility matrix | 维护 engine、model、quant、hardware、runtime 兼容矩阵 | matrix artifacts |
| Community recipe flow | recipe submission、validation、status lifecycle | contribution workflow |

### 9.4 验收标准

- 插件能被 application recipe 引用。
- RAG/Search pipeline 至少支持替换 embedding、parser、index。
- optimizer 能根据 benchmark 结果给出可解释调参建议。
- recipe 有 verified、experimental、deprecated、broken 状态流转。

---

## 10. 依赖优先级

### P0：第一阶段核心依赖

- vLLM
- SGLang
- llama.cpp
- Docker
- Docker Compose
- NVIDIA Container Toolkit
- Qdrant
- TEI 或 sentence-transformers
- Open WebUI，作为可选 UI

### P1：RAG/Search 支撑

- PyMuPDF
- Docling
- LlamaIndex / LangChain / Haystack，优先作为参考或 Expert backend
- FlagEmbedding / BGE reranker
- Ragas，放在系统性 RAG eval 阶段
- Elasticsearch / OpenSearch，若 Local Search 需要 keyword/hybrid search

### P2：Expert 长期优化

- FlashAttention
- FlashInfer
- Triton language
- TileLang
- ExLlamaV2
- ONNX Runtime
- OpenVINO
- ROCm stack
- Milvus / Weaviate
- DeepEval / promptfoo

---

## 11. 建议任务拆分顺序

### Sprint 1：Recipe 与 CLI 地基

- 建立 recipe 目录结构。
- 定义 inference/application recipe schema。
- 实现 recipe registry loader。
- 实现 `localaistack recipes list/validate`。
- 添加 2-3 个静态示例 recipe。

### Sprint 2：Runner、Preflight、Healthcheck

- 实现 Docker Compose runner。
- 实现 native runner skeleton。
- 实现基础 preflight。
- 实现 OpenAI-compatible healthcheck。
- 完成 vLLM 和 llama.cpp 的最小可运行 recipe。

### Sprint 3：Benchmark 与 Diagnostics

- 实现 simple benchmark。
- 采集 TTFT、tokens/s、RAM/VRAM、error rate。
- 建立 diagnostics rule table。
- 覆盖端口冲突、GPU 不可见、OOM、模型下载失败。

### Sprint 4：首批 Express Recipes 验证

- 验证 qwen14b-vllm-nvidia24g。
- 验证 qwen32b-awq-vllm-nvidia24g。
- 验证 CPU-only llama.cpp GGUF。
- 验证 Apple Metal llama.cpp GGUF。
- 验证 SGLang NVIDIA recipe。

### Sprint 5：Local Docs RAG

- 建立 application recipe schema。
- 实现 local folder ingestion。
- 支持 Markdown、TXT、PDF。
- 接入 embedding service 和 Qdrant。
- 实现 RAG API、citation、no-context fallback。

### Sprint 6：Selector

- 实现 hardware detector。
- 实现 intent schema。
- 实现 recipe match 和 confidence scoring。
- 实现 fallback planner。
- 打通 `express detect → recommend → run → healthcheck → benchmark`。

### Sprint 7：Expert Alpha

- 实现 expert input schema。
- 实现 model facts resolver。
- 实现 vLLM/SGLang/llama.cpp candidate generator。
- 输出 plan report 和 build artifacts。
- 接入 benchmark result store。

### Sprint 8：Plugin 与 Optimizer Beta

- 定义插件接口。
- 抽象 RAG/Search 插件。
- 引入 eval runner。
- 基于 benchmark 生成 tuning suggestions。
- 建立 compatibility matrix 和 community recipe workflow。

---

## 12. 风险与控制

| 风险 | 表现 | 控制策略 |
|---|---|---|
| 依赖版本漂移 | vLLM/SGLang/PyTorch/CUDA 组合失效 | 每个 verified recipe 固定版本，并维护 watch record |
| 首批覆盖过宽 | recipe 很多但不可用 | M1 限制在 5-10 个 verified recipes |
| GPU 环境差异 | Docker 看不到 GPU、driver 不匹配 | preflight 和 diagnostics 优先做扎实 |
| RAG 质量不稳定 | PDF 解析差、引用不准、上下文污染 | M2 先限制格式和 answering policy |
| Expert 过早复杂化 | compiler 规则不可维护 | M4 先 rule-based，基于 benchmark 数据再优化 |
| UI 投入过重 | 自研 UI 拖慢核心能力 | API-first，Open WebUI 可选 |

---

## 13. 成功指标

### Express Inference

- 首次启动成功率。
- 15 分钟内完成本地 API 启动比例。
- healthcheck 通过率。
- verified inference recipe 数量。
- OOM 后 fallback 成功比例。

### Express RAG/Search

- indexing 成功率。
- retrieval 命中率。
- citation 准确率。
- no-context fallback 准确率。
- answer faithfulness。

### Expert

- 动态生成 recipe 的可运行率。
- 推荐 engine 命中率。
- benchmark-driven tuning 后性能提升。
- plugin 覆盖度。
- 用户手动改配置次数下降。

---

## 14. 最近可执行的下一步

优先从 M0 + M1 的最小闭环开始：

```text
recipe schema
→ registry loader
→ CLI list/validate
→ one vLLM Docker recipe
→ one llama.cpp fallback recipe
→ preflight
→ healthcheck
→ benchmark
```

完成这个闭环后，再扩展 recipes 数量。这样可以避免先堆大量 compose 文件，却缺少统一验证、诊断和状态管理。
