# vLLM module

This module installs vLLM either from official CPU wheels (default) or from source using `uv`.

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

The source install expects `git` and `uv` to be available in `PATH` and uses
`VLLM_USE_PRECOMPILED=1 uv pip install --editable .` inside the cloned repo.
