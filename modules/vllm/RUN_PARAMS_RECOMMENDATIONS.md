# vLLM 运行参数建议

本文给出在 LocalAIStack 中使用 `vllm serve` 的参数建议，目标是优先稳定，再逐步提高吞吐。

特别说明：如果本机是 `Tesla V100 / SM70`，并且使用的是 `1CatAI/1Cat-vLLM` 分支，则应优先采用本文中的 V100 专项参数，而不是直接套用通用大显存卡模板。

## 1. 优先调参顺序

1. `--max-model-len`
2. `--gpu-memory-utilization`
3. `--max-num-seqs`
4. `--max-num-batched-tokens`
5. `--tensor-parallel-size`
6. `--dtype`
7. `--attention-backend` / `--compilation-config` / `--disable-custom-all-reduce`

## 2. 先看硬件分型

### 2.1 通用 CUDA 卡

- 适用范围：非 V100 的常规 CUDA 机器，或你不确定是否需要 `1Cat-vLLM` 分支。
- 起步思路：先用 `4096 + 0.88 + 4 seqs`，再按 OOM 或吞吐问题微调。

### 2.2 Tesla V100 / SM70

- 优先目标：稳定跑通 AWQ 4-bit，尤其是 `Qwen3 / Qwen3.5` 一类模型。
- 推荐基线来自 `1CatAI/1Cat-vLLM` README 在 `4 x Tesla V100-SXM2-16GB` 上的验证结果。
- 第一条真实请求可能需要 `1~3 分钟` 预热；这通常不是异常，而是编译 kernel、构建 cudagraph 和 warmup 的正常表现。

## 3. 快速起步模板

> 参数名请以 `vllm serve --help` 为准。

### 3.1 通用稳定模板（单卡）

```bash
vllm serve /path/to/model \
  --host 127.0.0.1 \
  --port 8000 \
  --dtype float16 \
  --max-model-len 4096 \
  --gpu-memory-utilization 0.88 \
  --tensor-parallel-size 1 \
  --max-num-seqs 4
```

### 3.2 通用长上下文模板（资源更高）

```bash
vllm serve /path/to/model \
  --host 127.0.0.1 \
  --port 8000 \
  --dtype float16 \
  --max-model-len 8192 \
  --gpu-memory-utilization 0.90 \
  --tensor-parallel-size 1 \
  --max-num-seqs 2
```

### 3.3 V100 / SM70 文本模型稳定模板

```bash
export VLLM_DISABLE_PYNCCL=1

vllm serve /path/to/model \
  --host 127.0.0.1 \
  --port 8000 \
  --dtype float16 \
  --tensor-parallel-size 4 \
  --attention-backend TRITON_ATTN \
  --disable-custom-all-reduce \
  --compilation-config '{"cudagraph_mode":"full_and_piecewise","cudagraph_capture_sizes":[1]}' \
  --gpu-memory-utilization 0.80 \
  --max-model-len 512 \
  --max-num-seqs 1 \
  --max-num-batched-tokens 128 \
  --skip-mm-profiling \
  --limit-mm-per-prompt '{"image":0,"video":0}'
```

### 3.4 V100 / SM70 视觉模型保守模板

```bash
export VLLM_DISABLE_PYNCCL=1

vllm serve /path/to/model \
  --host 127.0.0.1 \
  --port 8000 \
  --dtype float16 \
  --tensor-parallel-size 4 \
  --attention-backend TRITON_ATTN \
  --disable-custom-all-reduce \
  --compilation-config '{"cudagraph_mode":"full_and_piecewise","cudagraph_capture_sizes":[1]}' \
  --gpu-memory-utilization 0.80 \
  --max-model-len 4096 \
  --max-num-seqs 1 \
  --max-num-batched-tokens 512 \
  --limit-mm-per-prompt '{"image":1,"video":0}' \
  --allowed-local-media-path /path/to/media
```

## 4. 核心参数建议

### 4.1 `--max-model-len`

- 通用起点：`4096`
- 长上下文需求：`8192` 或更高
- V100 文本 AWQ 起点：`512`
- V100 视觉模型起点：`4096`
- 值越大，KV cache 和显存压力越高，吞吐通常下降

### 4.2 `--gpu-memory-utilization`

- 通用起点：`0.88`
- 通用常用范围：`0.85 ~ 0.93`
- V100 README 稳定基线：`0.80`
- 对 `4 x 16GB V100`，README 明确提到 `0.92` 虽然可能能启动，但首个真实请求不稳定

### 4.3 `--max-num-seqs`

- 通用延迟优先：`2 ~ 4`
- 通用吞吐优先：`8 ~ 16`，但需要足够显存
- V100 稳定起点：`1`
- 在 V100 小显存卡上，优先先把服务跑稳，再尝试从 `1` 增到 `2`

### 4.4 `--max-num-batched-tokens`

- 通用场景：如果没有明显瓶颈，可先让默认值工作
- V100 文本 AWQ 起点：`128`
- V100 视觉模型起点：`512`
- 如果首请求后仍频繁 OOM，和 `--max-model-len` 一起下调

### 4.5 `--tensor-parallel-size`

- 单卡：`1`
- 多卡：设置为 GPU 数或其因子，如 `2/4`
- V100 多卡 AWQ：优先按实卡数配置，例如 `4 x 16GB` 先用 `4`
- 配错会导致性能下降、NCCL 初始化失败或图捕获异常

### 4.6 `--dtype`

- 通用推荐：`float16` 或 `bfloat16`
- V100 / SM70 建议优先使用 `float16`
- 在 V100 专项模板里，不建议先从 `bfloat16` 起步

### 4.7 `--attention-backend`

- 通用场景：按 vLLM 默认值即可
- V100 / SM70：优先显式指定 `TRITON_ATTN`

### 4.8 `--compilation-config`

- 通用场景：先不主动覆盖
- V100 / SM70 推荐：

```text
--compilation-config '{"cudagraph_mode":"full_and_piecewise","cudagraph_capture_sizes":[1]}'
```

### 4.9 稳定性开关

- `--disable-custom-all-reduce`：V100 README 明确建议开启
- `VLLM_DISABLE_PYNCCL=1`：V100 README 明确建议开启
- `--enforce-eager`：只有在图编译或 cudagraph 相关异常时再尝试
- `--optimization-level`：通用机器可从低到高试；V100 分支优先先保持 README 的保守基线

## 5. 文本与多模态的分歧

### 5.1 文本-only

- V100 小显存卡建议加：
  - `--skip-mm-profiling`
  - `--limit-mm-per-prompt '{"image":0,"video":0}'`
- 这样可以避免多模态 profiling 占掉本就紧张的显存预算

### 5.2 视觉模型

- 不要传 `--skip-mm-profiling`
- 不要把 `--limit-mm-per-prompt` 设成 `{"image":0,"video":0}`
- 在 `4 x V100 16GB` 上，先从 `4096 / 512` 这种保守组合起步，再考虑往上提

## 6. 常见场景建议

### 6.1 首次上线

通用 CUDA 卡：

- `max-model-len=4096`
- `gpu-memory-utilization=0.88`
- `max-num-seqs=4`
- `tensor-parallel-size=1`

V100 / SM70：

- `VLLM_DISABLE_PYNCCL=1`
- `disable-custom-all-reduce=true`
- `attention-backend=TRITON_ATTN`
- `gpu-memory-utilization=0.80`
- `max-model-len=512`
- `max-num-seqs=1`
- `max-num-batched-tokens=128`

### 6.2 OOM 频繁

按顺序调整：

1. 降 `--max-model-len`
2. 降 `--max-num-batched-tokens`
3. 降 `--gpu-memory-utilization`
4. 降 `--max-num-seqs`

如果是 V100，优先不要一上来把 `gpu-memory-utilization` 拉高到 `0.9+`。

### 6.3 吞吐不足

通用机器按顺序调整：

1. 提高 `--max-num-seqs`
2. 多卡时提高 `--tensor-parallel-size`
3. 在不 OOM 前提下微升 `--gpu-memory-utilization`

V100 机器按顺序调整：

1. 先确认首个真实请求完成过预热，不要把 warmup 时间误判成稳态吞吐
2. 把 `--max-num-seqs` 从 `1` 试到 `2`
3. 视模型情况提高 `--max-num-batched-tokens`
4. 最后再小步上调 `--gpu-memory-utilization`

## 7. 与 LocalAIStack `smart-run` 配合建议

在 `model run --smart-run` 场景下，建议用户显式固定以下参数边界，避免漂移：

- `--max-model-len`
- `--gpu-memory-utilization`
- `--max-num-seqs`
- `--max-num-batched-tokens`
- `--tensor-parallel-size`

如果目标机器是 V100 / SM70，建议同时固定以下策略，不要让自动建议反复切换：

- `VLLM_DISABLE_PYNCCL=1`
- `--disable-custom-all-reduce`
- `--attention-backend TRITON_ATTN`
- 文本-only 时固定 `--skip-mm-profiling`
- 文本-only 时固定 `--limit-mm-per-prompt '{"image":0,"video":0}'`
