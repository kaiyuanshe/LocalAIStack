#!/usr/bin/env bash
set -euo pipefail

studio_home="${UNSLOTH_STUDIO_HOME:-${HOME}/.unsloth/studio}"

if [[ ! -d "$studio_home" ]]; then
  echo "Unsloth Studio home not found: $studio_home" >&2
  exit 1
fi

candidates=()
if command -v unsloth >/dev/null 2>&1; then
  candidates+=("$(command -v unsloth)")
fi
candidates+=("${studio_home}/unsloth_studio/bin/unsloth")

resolve_python_for_candidate() {
  local candidate="$1"
  local first_line=""
  if [[ ! -f "$candidate" ]]; then
    return 1
  fi

  IFS= read -r first_line < "$candidate" || true
  if [[ "$first_line" == '#!'* ]]; then
    printf '%s\n' "${first_line#\#!}"
    return 0
  fi

  local sibling_python
  sibling_python="$(dirname "$candidate")/python"
  if [[ -x "$sibling_python" ]]; then
    printf '%s\n' "$sibling_python"
    return 0
  fi

  return 1
}

for candidate in "${candidates[@]}"; do
  if [[ -x "$candidate" ]]; then
    python_bin="$(resolve_python_for_candidate "$candidate" || true)"
    if [[ -n "$python_bin" ]] && [[ -x "$python_bin" ]] && ! "$python_bin" - <<'PY' >/dev/null 2>&1
import importlib.util
import sys
raise SystemExit(0 if importlib.util.find_spec("_bz2") is not None else 1)
PY
    then
      echo "Unsloth Studio Python is missing _bz2: $python_bin" >&2
      exit 1
    fi
  fi

  if [[ -x "$candidate" ]] && "$candidate" studio --help >/dev/null 2>&1; then
    echo "Unsloth Studio verification succeeded: $candidate"
    exit 0
  fi
done

echo "Unsloth Studio is present under $studio_home, but the CLI could not be launched." >&2
exit 1
