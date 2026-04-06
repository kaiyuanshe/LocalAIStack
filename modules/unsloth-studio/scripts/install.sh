#!/usr/bin/env bash
set -euo pipefail

installer_url="${UNSLOTH_STUDIO_INSTALL_URL:-https://unsloth.ai/install.sh}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

python3 - <<'PY'
import importlib.util
import os
import sys

if importlib.util.find_spec("_bz2") is None:
    exe = sys.executable
    hint = [
        "Python is missing the standard-library _bz2 extension.",
        f"Current python3: {exe}",
        "Unsloth Studio depends on bz2 via the datasets package.",
    ]
    if ".pyenv/" in exe:
        hint.extend([
            "Detected a pyenv-managed Python.",
            "Install bzip2 development headers, reinstall that Python with pyenv, then retry.",
            "On Ubuntu/Debian: sudo apt-get install -y libbz2-dev",
            "Quick workaround: PYENV_VERSION=system ./build/las module install unsloth-studio",
        ])
    raise SystemExit("\n".join(hint))
PY

args=()
python_version="${UNSLOTH_STUDIO_PYTHON_VERSION:-3.12}"
if [[ -n "$python_version" ]]; then
  args+=(--python "$python_version")
fi
if [[ "${UNSLOTH_STUDIO_NO_TORCH:-0}" == "1" ]]; then
  args+=(--no-torch)
fi
if [[ "${UNSLOTH_STUDIO_VERBOSE:-0}" == "1" ]]; then
  args+=(--verbose)
fi

curl -fsSL "$installer_url" -o "$tmp_dir/install.sh"
sh "$tmp_dir/install.sh" "${args[@]}"
