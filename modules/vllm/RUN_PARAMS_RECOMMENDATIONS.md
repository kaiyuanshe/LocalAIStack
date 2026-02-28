# vLLM 运行参数建议

本文给出在 LocalAIStack 中使用 `vllm serve` 的参数建议，目标是优先稳定，再逐步提高吞吐。

## 1. 优先调参顺序

1. `--max-model-len`
2. `--gpu-memory-utilization`
3. `--tensor-parallel-size`
4. `--max-num-seqs`
5. `--dtype`
6. `--enforce-eager` / `--optimization-level` / `--disable-custom-all-reduce`

## 2. 快速起步模板

> 参数名请以 `vllm serve --help` 为准。

### 2.1 通用稳定模板（单卡）

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

### 2.2 长上下文模板（资源更高）

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

### 2.3 多卡模板（吞吐优先）

```bash
vllm serve /path/to/model \
  --host 127.0.0.1 \
  --port 8000 \
  --dtype bfloat16 \
  --max-model-len 4096 \
  --gpu-memory-utilization 0.90 \
  --tensor-parallel-size 2 \
  --max-num-seqs 8
```

## 3. 核心参数建议

### 3.1 `--max-model-len`

- 建议起点：`4096`
- 长上下文需求：`8192`（或更高）
- 值越大，显存占用越高，吞吐通常下降

### 3.2 `--gpu-memory-utilization`

- 建议起点：`0.88`
- 常用范围：`0.85 ~ 0.93`
- 如果频繁 OOM，先降到 `0.82 ~ 0.86`

### 3.3 `--tensor-parallel-size`

- 单卡：`1`
- 多卡：设置为 GPU 数或其因子（如 2/4）
- 配错会导致性能下降或初始化失败

### 3.4 `--max-num-seqs`

- 延迟优先：`2 ~ 4`
- 吞吐优先：`8 ~ 16`（需足够显存）
- 并发增大后，建议联动下调 `--max-model-len`

### 3.5 `--dtype`

- 推荐：`float16` 或 `bfloat16`
- 资源受限或兼容性问题时再考虑 `float32`（开销更高）

### 3.6 稳定性开关

- `--enforce-eager`：遇到图编译相关异常时可打开
- `--optimization-level`：建议从低到高逐步试（如 `0 -> 1 -> 2`）
- `--disable-custom-all-reduce`：多卡通信异常时可尝试开启

## 4. 常见场景建议

### 4.1 首次上线（稳）

- `max-model-len=4096`
- `gpu-memory-utilization=0.88`
- `max-num-seqs=4`
- `tensor-parallel-size=1`

### 4.2 OOM 频繁

按顺序调整：
1. 降 `--max-model-len`
2. 降 `--gpu-memory-utilization`
3. 降 `--max-num-seqs`

### 4.3 吞吐不足

按顺序调整：
1. 提高 `--max-num-seqs`
2. 多卡时提高 `--tensor-parallel-size`
3. 在不 OOM 前提下微升 `--gpu-memory-utilization`

## 5. 与 LocalAIStack `smart-run` 配合建议

在 `model run --smart-run` 场景下，建议用户显式固定以下参数边界，避免漂移：

- `--max-model-len`（业务最长输入上限）
- `--gpu-memory-utilization`（稳定区间）
- `--tensor-parallel-size`（按卡数固定）
- `--max-num-seqs`（按延迟或吞吐目标固定）

这样可以让 LLM 建议在安全范围内微调，而不是每次大幅波动。
