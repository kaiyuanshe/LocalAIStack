#!/usr/bin/env bash
set -euo pipefail

MODULE_HOME="${OBEAVER_HOME:-${HOME}/.localaistack/tools/obeaver}"
REPO_DIR="${MODULE_HOME}/repo"
VENV_DIR="${MODULE_HOME}/.venv"
REPO_URL="${OBEAVER_REPO_URL:-https://github.com/microsoft/obeaver.git}"
REPO_REF="${OBEAVER_REPO_REF:-main}"
LOCAL_BIN_DIR="${HOME}/.local/bin"
WRAPPER_PATH="${LOCAL_BIN_DIR}/obeaver"
PYTHON_BIN="${PYTHON_BIN:-python3}"

require_command() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Required command not found: $1" >&2
    exit 1
  }
}

ensure_foundry_local() {
  if command -v foundry >/dev/null 2>&1; then
    echo "Foundry Local already available: $(command -v foundry)"
    return 0
  fi

  case "$(uname -s)" in
    Darwin)
      if ! command -v brew >/dev/null 2>&1; then
        echo "Foundry Local is required on macOS for the default oBeaver engine, but Homebrew is not installed." >&2
        echo "Install Homebrew and rerun, or install Foundry Local manually with: brew install microsoft/foundrylocal/foundrylocal" >&2
        exit 1
      fi

      echo "Installing Foundry Local via Homebrew..."
      brew install microsoft/foundrylocal/foundrylocal
      command -v foundry >/dev/null 2>&1 || {
        echo "Foundry Local install completed, but 'foundry' is still not on PATH." >&2
        exit 1
      }
      ;;
    Linux)
      echo "Foundry Local is not supported on Linux. oBeaver will need to run with '--engine ort' and a local ONNX model directory." >&2
      ;;
    *)
      echo "Unsupported platform for automatic Foundry Local setup: $(uname -s)" >&2
      ;;
  esac
}

write_wrapper() {
  mkdir -p "$LOCAL_BIN_DIR"
  cat > "$WRAPPER_PATH" <<EOF
#!/usr/bin/env bash
exec "${VENV_DIR}/bin/obeaver" "\$@"
EOF
  chmod +x "$WRAPPER_PATH"
}

require_command git
require_command "$PYTHON_BIN"
ensure_foundry_local

mkdir -p "$MODULE_HOME"

if [[ ! -d "${REPO_DIR}/.git" ]]; then
  git clone --depth 1 --branch "$REPO_REF" "$REPO_URL" "$REPO_DIR"
else
  echo "oBeaver repository already present: ${REPO_DIR}"
fi

if [[ ! -x "${VENV_DIR}/bin/python" ]]; then
  "$PYTHON_BIN" -m venv "$VENV_DIR"
fi

"${VENV_DIR}/bin/python" -m pip install --upgrade pip setuptools wheel
"${VENV_DIR}/bin/python" -m pip install -e "$REPO_DIR"

write_wrapper

"${VENV_DIR}/bin/obeaver" version >/dev/null
echo "oBeaver installed at: ${WRAPPER_PATH}"
