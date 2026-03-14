---
name: localaistack-cli
description: Use this skill when a task involves inspecting, explaining, or operating the LocalAIStack CLI at ./build/las (command name localaistack), including model, module, provider, service, system, init, and failure subcommands.
---

# LocalAIStack CLI

Use this skill when the user asks to operate `./build/las`, explain its flags, debug a failing LocalAIStack CLI workflow, or map a task onto `localaistack` subcommands.

## Command identity

- The built binary is `./build/las`.
- Its Cobra command name is `localaistack`.
- Global flags:
  - `--config <path>` overrides the config file. Default path is `$HOME/.localaistack/config.yaml`.
  - `--verbose` enables verbose output.
- `localaistack init` and `localaistack system init` are the same initializer.

## Safe operating defaults

- Prefer `./build/las <command> --help` before guessing flags.
- Prefer read-only commands first: `model list`, `model search`, `module list`, `provider list`, `failure list`.
- For `model run`, use `--dry-run` first unless the user explicitly wants the process started.
- Do not use `model rm --force`, `module uninstall`, or `module purge` unless the user explicitly asked for deletion.
- If a command depends on machine state and fails, inspect `./build/las failure list` and `./build/las failure show <id>` before retrying blindly.

## Preconditions

- `module config-plan` and non-Ollama `model run` read base system info from `~/.localaistack/base_info.json`.
- If that file is missing, run `./build/las system init` or `./build/las init`.
- `system init` writes config to `~/.localaistack/config.yaml` and refreshes `base_info.json`.
- `model run --smart-run` also needs a valid LLM config in the user config file.

## Top-level commands

### `init` / `system init`

Use to bootstrap config and base system info.

Common non-interactive form:

```bash
./build/las system init \
  --api-key "$API_KEY" \
  --language en \
  --assistant-provider siliconflow \
  --assistant-model deepseek-ai/DeepSeek-V3.2
```

Important flags:

- `--config-path`
- `--api-key`
- `--language`
- `--assistant-provider`, `--assistant-model`, `--assistant-base-url`, `--assistant-timeout-seconds`
- `--translation-provider`, `--translation-model`, `--translation-base-url`, `--translation-timeout-seconds`

### `model`

Use for model discovery, download, local inventory, runtime launch, repair, and smart-run cache inspection.

Decision rules:

- `model search <query>` searches remote sources. Default `--source all`.
- `model download <model-id> [file]` downloads a model. `--file` and positional `[file]` are mutually exclusive.
- `model list` lists locally downloaded models under `~/.localaistack/models`.
- `model repair <model-id>` repairs missing config/tokenizer files for Hugging Face or ModelScope models.
- `model rm <model-id> --force` deletes a local model.
- `model run <model-id>` starts a runtime for a local model.

Source rules:

- Explicit `--source` accepts `ollama`, `huggingface` or `hf`, and `modelscope`.
- Without `--source`, the CLI tries to parse the source from the model id.

Runtime rules for `model run`:

- Ollama source: shells out to `ollama run <model>`.
- Local model directory with `.safetensors`: uses `vllm`.
- Local model directory with `.gguf`: uses `llama.cpp`.
- If both safetensors and gguf exist, safetensors/vLLM path is chosen first.

Practical patterns:

```bash
./build/las model search qwen3 --source all --limit 5
./build/las model download hf/Qwen/Qwen3-8B
./build/las model run hf/Qwen/Qwen3-8B --dry-run
./build/las model run hf/bartowski/DeepSeek-R1-Distill-Qwen-7B-GGUF Q4_K_M --dry-run
./build/las model repair hf/Qwen/Qwen3-8B
./build/las model smart-run-cache list --runtime vllm
```

Important `model run` flags:

- Common: `--dry-run`, `--source`, `--file`, `--host`, `--port`
- Smart planning: `--smart-run`, `--smart-run-debug`, `--smart-run-refresh`, `--smart-run-strict`
- llama.cpp: `--threads`, `--ctx-size`, `--n-gpu-layers`, `--tensor-split`, `--batch-size`, `--ubatch-size`, `--auto-batch`, `--temperature`, `--top-p`, `--top-k`, `--min-p`, `--presence-penalty`, `--repeat-penalty`, `--chat-template-kwargs`
- vLLM: `--vllm-max-model-len`, `--vllm-gpu-memory-utilization`, `--vllm-trust-remote-code`, `--text-only`

Failure handling for `model run`:

- Missing `ollama` in `PATH`: install the Ollama module first.
- Missing `vllm` in `PATH`: install the `vllm` module first.
- Missing base info: run `./build/las system init`.
- For repeated smart-run issues, inspect and clear cache entries with `model smart-run-cache`.

### `module`

Use for software module lifecycle and configuration planning.

Commands:

- `module list`
- `module check <module>`
- `module install <module>`
- `module update <module>`
- `module uninstall <module>`
- `module purge <module>`
- `module setting <module> <args...>`
- `module config-plan <module>`

`module config-plan` notes:

- Reads base info from `~/.localaistack/base_info.json`.
- `--model <id>` adds model context.
- `--dry-run` prints the plan without saving.
- `--apply` saves the plan to `~/.localaistack/config-plans/<module>.json`.
- `--output json` is best when another agent needs structured output.
- `--planner-strict` fails if planning cannot produce a valid plan.

Recommended flow:

```bash
./build/las module list
./build/las module check vllm
./build/las module config-plan vllm --output json --dry-run
./build/las module install vllm
```

### `provider`

Use `./build/las provider list` to list built-in LLM providers. This is read-only.

### `service`

Use for service lifecycle:

- `service start <name>`
- `service stop <name>`
- `service status <name>`

Treat these as stateful operations. Check status before and after changes.

### `system`

Current behavior:

- `system init` is real and writes config plus base info.
- `system detect` currently only prints `Detecting hardware...`.
- `system info` currently only prints `System information:`.

Do not assume `system detect` or `system info` return structured hardware data unless the code changes.

### `failure`

Use to inspect recorded failures after planner, install, or run problems.

Commands:

- `failure list --limit 20`
- `failure list --phase model_run --output json`
- `failure show <event-id>`

`failure show` returns JSON with both the event and suggested advice.

## Agent workflow

1. Confirm whether the task is read-only, mutating, or destructive.
2. If the task touches planning or runtime, ensure config and base info exist.
3. Use `--help` or read the command source before assuming unsupported flags.
4. Prefer `--dry-run` or JSON/text inspection before launching long-running services.
5. After failures, inspect `failure list` / `failure show` and only then retry with adjusted flags.

## Source files worth checking when behavior is unclear

- `internal/cli/root.go`
- `internal/cli/commands/commands.go`
- `internal/cli/commands/init.go`
- `internal/cli/commands/failure_commands.go`
