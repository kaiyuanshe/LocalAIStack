# Smart Run Model (P2) Design and Runbook

## 1. Scope

This document describes P2 smart runtime tuning for:

1. `llama.cpp` model serving
2. `vLLM` model serving

P2 covers:

1. static baseline parameter planning
2. optional LLM advice overlays
3. debug and strict controls
4. dry-run verification workflow

---

## 2. Model Role

Smart run uses `llm.model` from config.

Recommended default:

1. `llm.model: deepseek-ai/DeepSeek-V3.2`

Translation model configuration is not used in smart run.

---

## 3. CLI Controls

`model run` supports the following P2 controls:

1. `--smart-run` enables LLM advice
2. `--smart-run-debug` prints planner source and fallback reason
3. `--smart-run-strict` fails when advice cannot be obtained or parsed
4. `--dry-run` prints final command and exits without launching runtime

---

## 4. Parameter Precedence

P2 applies parameters in this order:

1. explicit user flags
2. LLM advice (`--smart-run`)
3. static defaults and auto-tuning

This guarantees user intent is never silently overridden.

---

## 5. Runtime Flow

For `./build/las model run <model-id>`:

1. resolve model source and local path
2. detect runtime type (`safetensors` -> `vLLM`, `GGUF` -> `llama.cpp`)
3. compute static baseline parameters
4. if `--smart-run`, request LLM advice using baseline + hardware context
5. validate and clamp advice values to safe ranges
6. merge values according to precedence
7. print or execute final command

---

## 6. Debug and Strict Semantics

Debug output:

1. `source=llm` means advice was accepted
2. `source=static` means fallback path was used
3. `reason=...` explains success or fallback reason

Strict behavior:

1. if `--smart-run-strict` is set without `--smart-run`, command fails
2. if smart-run advice fails and strict is on, command fails
3. if strict is off, command falls back to static path

---

## 7. Recommended Commands

Baseline verification:

```bash
./build/las model run <model-id> --smart-run --smart-run-debug --dry-run
```

Strict validation (LLM must succeed):

```bash
./build/las model run <model-id> --smart-run --smart-run-debug --smart-run-strict --dry-run
```

GGUF tuning with batch auto-tune:

```bash
./build/las model run <model-id> --smart-run --smart-run-debug --auto-batch
```

---

## 8. Proving DeepSeek Participation

To prove `deepseek-ai/DeepSeek-V3.2` is involved in run planning:

1. set `llm.model=deepseek-ai/DeepSeek-V3.2`
2. run with `--smart-run --smart-run-debug --smart-run-strict`
3. confirm planner output shows `source=llm`
4. verify provider-side logs by model and timestamp

If output shows `source=static`, run planning did not use LLM advice for that run.

---

## 9. Known Limits (P2)

1. planner traces are console-only (no persistent run trace store yet)
2. advice schema is runtime-specific and currently limited to known safe fields
3. no automatic online troubleshooting in P2

These are addressed by later phases.
