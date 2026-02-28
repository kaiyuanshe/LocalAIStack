# LocalAIStack

最简使用步骤：下载代码 → 编译 → 使用 `./build/las`。

## 1) 下载代码

```bash
git clone <repo-url> LocalAIStack
cd LocalAIStack
```

## 2) 编译

```bash
make tidy
make build
```

编译产物：`./build/las`（CLI）与 `./build/las-server`（服务端）。

## 3) 使用 `./build/las`

你可以先看总览：

```bash
./build/las --help
```

### 3.1 初始化与系统信息

```bash
# 交互式初始化（推荐）
./build/las init

# 非交互初始化（可脚本化）
./build/las init \
  --api-key "sk-xxxx" \
  --language zh-CN \
  --assistant-provider siliconflow \
  --assistant-model deepseek-ai/DeepSeek-V3.2 \
  --translation-provider siliconflow \
  --translation-model tencent/Hunyuan-MT-7B

# 系统信息与硬件检测
./build/las system info
./build/las system detect

# system init 与 init 等价
./build/las system init
```

效果：
* 生成配置文件：`$HOME/.localaistack/config.yaml`
* 默认双模型职责：
  * 翻译模型：`tencent/Hunyuan-MT-7B`
  * 智能助手模型：`deepseek-ai/DeepSeek-V3.2`（可修改）
* 生成硬件基础信息（`base_info.json`），用于 install planner / config planner / smart-run

### 3.2 模块管理（module）

```bash
# 查看可管理模块
./build/las module list

# 安装 / 升级 / 卸载 / 清理
./build/las module install ollama
./build/las module update ollama
./build/las module uninstall ollama
./build/las module purge ollama

# 健康检查
./build/las module check ollama

# 模块特定设置
./build/las module setting comfyui Comfy-Org_z_image_turbo
```

### 3.3 安装后配置规划（module config-plan）

```bash
# 为模块生成配置计划（文本）
./build/las module config-plan llama.cpp

# JSON 输出
./build/las module config-plan llama.cpp --output json

# 仅演示，不落盘
./build/las module config-plan vllm --dry-run --planner-debug

# 严格模式：如果 planner 失败则直接报错
./build/las module config-plan vllm --planner-strict --dry-run

# 落盘到 ~/.localaistack/config-plans/<module>.json
./build/las module config-plan ollama --apply
```

### 3.4 模型管理（model）

```bash
# 搜索模型（支持 all / ollama / huggingface / modelscope）
export HF_ENDPOINT=https://hf-mirror.com
./build/las model search qwen3
./build/las model search qwen3 --source huggingface --limit 20

# 下载模型
./build/las model download qwen3-coder:30b
./build/las model download unsloth/Qwen3-Coder-Next-GGUF
./build/las model download unsloth/Qwen3-Coder-Next-GGUF --file Q4_K_M.gguf

# 列出已下载模型
./build/las model list

# 修复模型缺失的 tokenizer/config 文件
./build/las model repair ByteDance/Ouro-2.6B-Thinking

# 删除模型
./build/las model rm qwen3-coder:30b --force
```

### 3.5 运行模型（model run）

```bash
# 运行 GGUF（llama.cpp）
./build/las model run unsloth/Qwen3-Coder-Next-GGUF

# 指定运行参数
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --ctx-size 65536 \
  --threads 16 \
  --n-gpu-layers 40

# 自动 batch 调优 + 仅打印命令
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --auto-batch --dry-run

# smart-run（使用 LLM 做参数建议）
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --smart-run --smart-run-debug --dry-run

# smart-run 严格模式
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --smart-run --smart-run-strict --dry-run

# 运行 safetensors（vLLM）
./build/las model run ByteDance/Ouro-2.6B-Thinking \
  --vllm-max-model-len 8192 \
  --vllm-gpu-memory-utilization 0.9
```

### 3.6 Provider 与服务管理

```bash
# 查看可用 LLM provider
./build/las provider list

# 服务管理
./build/las service start ollama
./build/las service status ollama
./build/las service stop ollama
```

### 3.7 失败处理闭环（failure）

```bash
# 列出最近失败记录
./build/las failure list --limit 20

# 按阶段或分类过滤
./build/las failure list --phase smart_run --category timeout

# JSON 输出
./build/las failure list --output json --limit 5

# 查看单条失败详情与修复建议
./build/las failure show <event-id>
```

可选调试开关：

```bash
export LOCALAISTACK_FAILURE_DEBUG=1
```

失败时会额外输出：`phase/category/retryable/log_path/suggestion`。

### 3.8 Shell 补全

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

### 3.9 全局参数

```bash
# 使用指定配置文件
./build/las --config /tmp/las-config.yaml module list

# 打开 verbose 输出
./build/las --verbose model list
```
