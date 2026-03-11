# vLLM module

This module installs vLLM either from official CPU wheels (default) or from source using `uv`.

When `bash modules/vllm/scripts/install.sh` detects a local `Tesla V100` GPU, it delegates to `bash modules/vllm/scripts/install_for_v100.sh` and installs the `1CatAI/1Cat-vLLM` fork recommended for SM70/V100.

## Default (CPU wheel)

```bash
bash modules/vllm/scripts/install.sh
```

Environment variables:

- `VLLM_VERSION`: pin a release version (e.g. `0.6.3`).
- `VLLM_WHEEL_URL`: full URL to a wheel to install.
- `VLLM_PYTHON`: override the Python executable (default: `python3`).

## Source install (matches the vLLM docs flow)

```bash
VLLM_INSTALL_METHOD=source bash modules/vllm/scripts/install.sh
```

Environment variables:

- `VLLM_INSTALL_METHOD=source`: enable the source install path.
- `VLLM_SOURCE_DIR`: local path to clone into (default: `~/vllm`).
- `VLLM_REPO_URL`: git repo URL (default: `https://github.com/vllm-project/vllm.git`).
- `VLLM_PYTHON_VERSION`: Python version for `uv venv` (default: `3.12`).
- `VLLM_PRECOMPILED_WHEEL_COMMIT`: override the precompiled-wheel commit lookup. If unset, the installer retries with `nightly` when the current `main` commit has no published metadata yet.

The source install expects `git` and `uv` to be available in `PATH` and uses
`VLLM_USE_PRECOMPILED=1 uv pip install --editable .` inside the cloned repo.

## Tesla V100 install

The V100-specific installer follows the `1CatAI/1Cat-vLLM` README flow:

```bash
bash modules/vllm/scripts/install_for_v100.sh
```

Environment variables:

- `VLLM_V100_SOURCE_DIR`: local path to clone into (default: `~/1Cat-vLLM`).
- `VLLM_V100_REPO_URL`: git repo URL (default: `https://github.com/1CatAI/1Cat-vLLM.git`).
- `VLLM_PYTHON_VERSION`: Python version for `uv venv` (default: `3.12`).
- `VLLM_V100_TORCH_INDEX_URL`: PyTorch wheel index URL (default: `https://download.pytorch.org/whl/cu128`).
- `VLLM_V100_TORCH_PACKAGES`: torch packages to install before running `use_existing_torch.py`.
- `TORCH_CUDA_ARCH_LIST`: CUDA arch list for V100 builds (default: `7.0`).
- `VLLM_FORCE_V100_INSTALL=1`: force the V100 path even if auto-detection fails.
- `VLLM_FORCE_GENERIC_INSTALL=1`: skip the V100 path and use the generic installer.
