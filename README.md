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

```bash
./build/las init
```

效果：
* 创建一个配置文件： `$HOME/.localaistack/config.yaml`
* 输入：SiliconFlow API Key，系统默认使用两个模型：
  * 翻译模型：`tencent/Hunyuan-MT-7B`
  * 智能助手模型：`deepseek-ai/DeepSeek-V3.2`（可修改）
* 输入：首选语言，你可以输入zh-CN，或者任何你喜欢的语言。

```bash
./build/las module list
```

效果：列出LocalAIStack能够管理的软件。

```bash
./build/las module install ollama
```

效果：安装Ollama

```bash
export HF_ENDPOINT=https://hf-mirror.com
./build/las model search qwen3
```

效果：搜索qwen3系列模型


```bash
./build/las model download qwen3-coder:30b
# or
./build/las model download unsloth/Qwen3-Coder-Next-GGUF
# or
./build/las model download unsloth/Qwen3-Coder-Next-GGUF Q4_K_M
```

效果：从ollama或huggingface下载模型（可以指定尺寸）

```bash
./build/las model run unsloth/Qwen3-Coder-Next-GGUF
# or
./build/las model run unsloth/Qwen3-Coder-Next-GGUF --ctx-size 65536
```

效果：用llama.cpp启动一个模型

```bash
./build/las model search ByteDance
./build/las model download ByteDance/Ouro-2.6B-Thinking
./build/las model run ByteDance/Ouro-2.6B-Thinking
```

效果：用vllm启动一个safetensors格式的模型。

```bash
./build/las model download Comfy-Org/z_image_turbo
./build/las module install comfyui
./build/las module setting comfyui Comfy-Org_z_image_turbo
comfyui-las --listen 0.0.0.0 --port 8188
```

效果：安装并启动ComfyUI，并使用z_image_turbo模型。
需要下载配套的[JSON文件](https://raw.githubusercontent.com/Comfy-Org/workflow_templates/refs/heads/main/templates/image_z_image_turbo.json)
