#!/usr/bin/env bash
set -euo pipefail

check_python_env() {
  local venv_dir="$1"
  local python_bin="$venv_dir/bin/python"

  [[ -x "$python_bin" ]] || return 1

  "$python_bin" - <<'PY'
import vllm
print(vllm.__version__)
PY
}

check_vllm_cli() {
  local vllm_bin=""

  vllm_bin="$(command -v vllm 2>/dev/null || true)"
  [[ -n "$vllm_bin" ]] || return 1

  "$vllm_bin" --version
}

main() {
  local -a candidate_venvs=(
    "${VLLM_VENV_DIR:-$HOME/.localaistack/venv/vllm}"
    "${VLLM_SOURCE_DIR:-$HOME/vllm}/.venv"
    "${VLLM_V100_SOURCE_DIR:-$HOME/1Cat-vLLM}/.venv"
    "$HOME/.venv/vllm"
  )
  local candidate=""

  for candidate in "${candidate_venvs[@]}"; do
    if check_python_env "$candidate"; then
      return 0
    fi
  done

  if check_vllm_cli; then
    return 0
  fi

  if python3 - <<'PY'
import importlib.util
import sys
sys.exit(0 if importlib.util.find_spec("vllm") else 1)
PY
  then
    python3 - <<'PY'
import vllm
print(vllm.__version__)
PY
    return 0
  fi

  echo "vLLM not found. Checked: ${candidate_venvs[*]} and PATH entry 'vllm'." >&2
  return 1
}

main "$@"
