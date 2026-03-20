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

# 例如安装训练/微调框架 Unsloth
./build/las module install unsloth
./build/las module check unsloth

# 模块特定设置
./build/las module setting comfyui Comfy-Org_z_image_turbo
```

当前仓库内已接入的模块包含推理运行时（如 `ollama`、`llama.cpp`、`vllm`）、模型工具（如 `hf`、`modelscope`）以及训练/微调框架 `unsloth`。`unsloth` 按官方默认 Linux 安装路径使用 `python3 -m pip install --user unsloth`，并要求 `Python < 3.14`。

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

# smart-run 强制刷新（忽略本地缓存，重新询问 LLM）
./build/las model run unsloth/Qwen3-Coder-Next-GGUF \
  --smart-run --smart-run-refresh --smart-run-debug --dry-run

# 查看 smart-run 缓存
./build/las model smart-run-cache list
./build/las model smart-run-cache list unsloth/Qwen3-Coder-Next-GGUF

# 删除某个模型的 smart-run 缓存
./build/las model smart-run-cache rm unsloth/Qwen3-Coder-Next-GGUF

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

### 3.10 命令总览（按命令树梳理）

下面这份树形清单对应当前 CLI 已注册的命令入口，适合快速查找：

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

补充说明：
* `init` 同时注册在根命令和 `system init` 下，两者等价
* `module` 还有别名：`modules`
* `model repair` 还有别名：`model fix`

### 3.11 命令参考（用途、参数、常用标志）

### 3.12 速查表（命令 / 用途 / 常用示例）

| 命令 | 用途 | 常用示例 |
| --- | --- | --- |
| `./build/las init` | 初始化用户配置与基础硬件信息 | `./build/las init --language zh-CN --api-key "sk-xxxx"` |
| `./build/las system init` | 与 `init` 等价的系统初始化入口 | `./build/las system init` |
| `./build/las system detect` | 触发硬件检测入口 | `./build/las system detect` |
| `./build/las system info` | 查看系统信息入口 | `./build/las system info` |
| `./build/las module list` | 列出可管理模块 | `./build/las module list` |
| `./build/las module install <module>` | 安装模块 | `./build/las module install ollama` |
| `./build/las module update <module>` | 升级模块 | `./build/las module update llama.cpp` |
| `./build/las module uninstall <module>` | 卸载模块 | `./build/las module uninstall vllm` |
| `./build/las module purge <module>` | 深度清理模块 | `./build/las module purge ollama` |
| `./build/las module check <module>` | 检查模块状态 | `./build/las module check comfyui` |
| `./build/las module setting <module> ...` | 运行模块特定设置 | `./build/las module setting comfyui Comfy-Org_z_image_turbo` |
| `./build/las module config-plan <module>` | 生成模块配置规划 | `./build/las module config-plan vllm --planner-debug --dry-run` |
| `./build/las service start <service>` | 启动服务 | `./build/las service start ollama` |
| `./build/las service stop <service>` | 停止服务 | `./build/las service stop ollama` |
| `./build/las service status <service>` | 查看服务状态 | `./build/las service status ollama` |
| `./build/las provider list` | 列出内置 LLM provider | `./build/las provider list` |
| `./build/las model search <query>` | 搜索模型 | `./build/las model search qwen3 --source huggingface --limit 20` |
| `./build/las model download <model-id>` | 下载模型 | `./build/las model download unsloth/Qwen3-Coder-Next-GGUF --file Q4_K_M.gguf` |
| `./build/las model list` | 列出已下载模型 | `./build/las model list` |
| `./build/las model repair <model-id>` | 修复模型支持文件 | `./build/las model repair ByteDance/Ouro-2.6B-Thinking` |
| `./build/las model rm <model-id>` | 删除模型 | `./build/las model rm qwen3-coder:30b --force` |
| `./build/las model run <model-id>` | 启动本地模型 | `./build/las model run unsloth/Qwen3-Coder-Next-GGUF --ctx-size 65536 --threads 16` |
| `./build/las model run <model-id> --auto-batch` | 自动调优 batch/ubatch | `./build/las model run unsloth/Qwen3-Coder-Next-GGUF --auto-batch --dry-run` |
| `./build/las model run <model-id> --smart-run` | 用 smart-run 自动建议运行参数 | `./build/las model run unsloth/Qwen3-Coder-Next-GGUF --smart-run --smart-run-debug` |
| `./build/las model run <model-id> --smart-run-refresh` | 忽略缓存并强制重新询问 LLM | `./build/las model run unsloth/Qwen3-Coder-Next-GGUF --smart-run --smart-run-refresh --dry-run` |
| `./build/las model smart-run-cache list` | 列出 smart-run 缓存 | `./build/las model smart-run-cache list unsloth/Qwen3-Coder-Next-GGUF` |
| `./build/las model smart-run-cache rm <model-id>` | 删除某模型的 smart-run 缓存 | `./build/las model smart-run-cache rm unsloth/Qwen3-Coder-Next-GGUF` |
| `./build/las failure list` | 列出失败记录 | `./build/las failure list --phase smart_run --category timeout` |
| `./build/las failure show <event-id>` | 查看单条失败详情与建议 | `./build/las failure show evt-xxxx` |

#### 根命令

* `./build/las --help`
  * 查看完整帮助
* 全局标志：
  * `--config <path>`：指定配置文件路径
  * `--verbose`：打开详细输出

#### `init` / `system init`

用途：
* 初始化用户配置
* 写入 `~/.localaistack/config.yaml`
* 生成 `~/.localaistack/base_info.json`

常用标志：
* `--config-path <path>`：指定配置文件输出位置
* `--api-key <key>`：设置 SiliconFlow API key
* `--language <lang>`：设置交互语言，例如 `zh-CN`、`en`
* `--assistant-provider <name>` / `--assistant-model <id>`：设置智能助手模型
* `--assistant-base-url <url>` / `--assistant-timeout-seconds <n>`：设置智能助手访问参数
* `--translation-provider <name>` / `--translation-model <id>`：设置翻译模型
* `--translation-base-url <url>` / `--translation-timeout-seconds <n>`：设置翻译访问参数

#### `module`

用途：
* 管理软件模块（安装、升级、卸载、清理、健康检查、参数设置）

子命令：
* `module list`
  * 列出仓库内可管理模块，并显示是否已安装
* `module install <module>`
  * 安装模块
* `module update <module>`
  * 升级模块
* `module uninstall <module>`
  * 卸载模块
* `module purge <module>`
  * 深度清理模块
* `module check <module>`
  * 校验模块是否可用
* `module setting <module> <setting-args...>`
  * 调用模块自定义设置逻辑
* `module config-plan <module>`
  * 为模块生成配置规划

`module config-plan` 常用标志：
* `--model <model-id>`：将模型 ID 一起纳入规划上下文
* `--apply`：保存到 `~/.localaistack/config-plans/<module>.json`
* `--dry-run`：仅输出，不保存
* `--planner-debug`：打印规划来源与原因
* `--planner-strict`：规划失败时直接报错
* `--output text|json`：设置输出格式

#### `service`

用途：
* 统一服务生命周期入口

子命令：
* `service start <service>`
* `service stop <service>`
* `service status <service>`

适合管理由 LocalAIStack 接管的后台服务，例如 `ollama`。

#### `model`

用途：
* 搜索、下载、列出、运行、修复、删除模型

子命令：
* `model search <query>`
  * 搜索模型仓库
  * 标志：
    * `--source, -s all|ollama|huggingface|modelscope`
    * `--limit, -n <N>`
* `model download <model-id> [file]`
  * 下载模型，可选指定单个文件
  * 标志：
    * `--source, -s <source>`
    * `--file, -f <filename>`
* `model list`
  * 列出已下载模型
* `model rm <model-id>`
  * 删除模型
  * 标志：
    * `--force, -f`
    * `--source, -s <source>`
* `model repair <model-id>`
  * 补齐 tokenizer/config 等支持文件
  * 标志：
    * `--source, -s <source>`
* `model run <model-id> [gguf-file-or-quant]`
  * 启动本地模型
  * 运行时自动区分：
    * `GGUF` → `llama.cpp`
    * `safetensors` → `vLLM`

`model run` 常用标志：
* 模型来源与文件选择：
  * `--source, -s <source>`
  * `--file, -f <gguf-file>`
* llama.cpp 推理参数：
  * `--threads`
  * `--ctx-size`
  * `--n-gpu-layers`
  * `--tensor-split`
  * `--batch-size`
  * `--ubatch-size`
  * `--auto-batch`
* 采样参数：
  * `--temperature`
  * `--top-p`
  * `--top-k`
  * `--min-p`
  * `--presence-penalty`
  * `--repeat-penalty`
  * `--chat-template-kwargs`
* vLLM 参数：
  * `--vllm-max-model-len`
  * `--vllm-gpu-memory-utilization`
  * `--vllm-trust-remote-code`
* 通用运行参数：
  * `--host`
  * `--port`
  * `--dry-run`

`model run` 中与 smart-run 相关的标志：
* `--smart-run`
  * 启用基于硬件/模型上下文的智能参数建议
* `--smart-run-debug`
  * 输出参数来源与回退原因
* `--smart-run-refresh`
  * 忽略本地已缓存的 smart-run 参数，强制重新询问 LLM
* `--smart-run-strict`
  * 如果 smart-run 失败，则命令直接失败

smart-run 参数持久化：
* 缓存目录：`~/.localaistack/smart-run/`
* 保存时机：模型进程成功启动后立即保存
* 优先级顺序：
  * 用户显式传参
  * 本地已保存 smart-run 参数
  * 新鲜 LLM 建议
  * 静态默认值 / auto-tune

#### `model smart-run-cache`

用途：
* 查看和删除已持久化的 smart-run 参数

子命令：
* `model smart-run-cache list [model-id]`
  * 列出全部缓存，或仅列某个模型
  * 标志：
    * `--runtime llama.cpp|vllm`
* `model smart-run-cache rm <model-id>`
  * 删除某个模型的缓存
  * 标志：
    * `--runtime llama.cpp|vllm`

#### `provider`

用途：
* 查看内置 LLM provider

子命令：
* `provider list`

#### `failure`

用途：
* 查看失败记录与建议动作

子命令：
* `failure list`
  * 列出最近失败记录
  * 标志：
    * `--limit <N>`
    * `--phase <phase>`
    * `--category <category>`
    * `--output text|json`
* `failure show <event-id>`
  * 查看单条失败详情和建议

常见 `phase`：
* `install_planner`
* `config_planner`
* `smart_run`
* `module_install`
* `model_run`

#### `system`

用途：
* 系统初始化与信息入口

子命令：
* `system init`
  * 与根命令 `init` 等价
* `system detect`
  * 硬件检测入口
* `system info`
  * 系统信息入口
