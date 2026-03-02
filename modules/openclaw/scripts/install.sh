#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

NVM_DIR="${NVM_DIR:-${HOME}/.nvm}"
OPENCLAW_NODE_MIN_MAJOR="${OPENCLAW_NODE_MIN_MAJOR:-22}"
OPENCLAW_NODE_VERSION="${OPENCLAW_NODE_VERSION:-lts/*}"
NVM_INSTALL_VERSION="${NVM_INSTALL_VERSION:-v0.40.3}"

ensure_usr_local_bin() {
  $SUDO mkdir -p /usr/local/bin
}

has_openclaw() {
  command -v openclaw >/dev/null 2>&1
}

load_nvm() {
  if [[ -s "${NVM_DIR}/nvm.sh" ]]; then
    # shellcheck disable=SC1090
    source "${NVM_DIR}/nvm.sh"
    command -v nvm >/dev/null 2>&1
    return $?
  fi

  return 1
}

install_nvm() {
  curl -fsSL "https://raw.githubusercontent.com/nvm-sh/nvm/${NVM_INSTALL_VERSION}/install.sh" | bash
  load_nvm
}

node_major_version() {
  local node_version
  node_version="$(node -v 2>/dev/null || true)"
  node_version="${node_version#v}"
  printf '%s\n' "${node_version%%.*}"
}

is_node_version_compatible() {
  if ! command -v node >/dev/null 2>&1; then
    return 1
  fi

  local major
  major="$(node_major_version)"

  [[ "$major" =~ ^[0-9]+$ ]] || return 1
  (( major >= OPENCLAW_NODE_MIN_MAJOR ))
}

ensure_nodejs_for_npm() {
  load_nvm || install_nvm || return 1

  # Prefer nvm default when available so we don't stay on system node.
  if [[ "$(nvm version default 2>/dev/null || true)" != "N/A" ]]; then
    nvm use default >/dev/null 2>&1 || true
  fi

  if is_node_version_compatible && command -v npm >/dev/null 2>&1 && [[ "$(command -v node)" == "${NVM_DIR}"/* ]]; then
    return 0
  fi

  nvm install "$OPENCLAW_NODE_VERSION"
  nvm use "$OPENCLAW_NODE_VERSION"

  if ! is_node_version_compatible || ! command -v npm >/dev/null 2>&1; then
    return 1
  fi

  return 0
}

install_from_script() {
  local install_url="$1"
  local tmp_dir
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' RETURN

  curl -fsSL "$install_url" -o "$tmp_dir/openclaw_install.sh"
  $SUDO bash "$tmp_dir/openclaw_install.sh"
}

install_from_npm() {
  ensure_nodejs_for_npm || return 1

  local npm_pkgs=(
    "openclaw"
    "@openclaw/cli"
    "openclaw-cli"
  )

  local pkg
  for pkg in "${npm_pkgs[@]}"; do
    if [[ "$(command -v npm)" == "${NVM_DIR}"/* ]]; then
      npm install -g "$pkg" && return 0
    else
      $SUDO npm install -g "$pkg" && return 0
    fi
  done

  return 1
}

ensure_python_openclaw_wrapper() {
  has_openclaw && return 0

  command -v python3 >/dev/null 2>&1 || return 1
  python3 -c 'import importlib.util,sys; sys.exit(0 if importlib.util.find_spec("openclaw") else 1)' || return 1

  local user_bin="${HOME}/.local/bin"
  mkdir -p "$user_bin"

  cat > "${user_bin}/openclaw" <<'EOF'
#!/usr/bin/env bash
exec python3 -m openclaw "$@"
EOF
  chmod +x "${user_bin}/openclaw"

  [[ -x "${user_bin}/openclaw" ]]
}

install_from_pip() {
  if command -v pipx >/dev/null 2>&1; then
    pipx install openclaw || pipx upgrade openclaw || true
    has_openclaw && return 0
  fi

  if command -v python3 >/dev/null 2>&1; then
    python3 -m pip install --user --upgrade openclaw
    has_openclaw && return 0
    ensure_python_openclaw_wrapper && return 0
  fi

  return 1
}

promote_binary_to_usr_local() {
  if has_openclaw; then
    return 0
  fi

  local candidates=(
    "${HOME}/.local/bin/openclaw"
    "/root/.local/bin/openclaw"
  )

  for candidate in "${candidates[@]}"; do
    if [[ -x "$candidate" ]]; then
      ensure_usr_local_bin
      $SUDO install -m 0755 "$candidate" /usr/local/bin/openclaw
      return 0
    fi
  done

  return 1
}

main() {
  local install_url="${OPENCLAW_INSTALL_URL:-https://openclaw.ai/install.sh}"

  if has_openclaw; then
    echo "OpenClaw already installed: $(command -v openclaw)"
    return 0
  fi

  if ! has_openclaw; then
    install_from_npm || true
  fi

  if ! has_openclaw; then
    if install_from_script "$install_url" 2>/dev/null || true; then
      :
    fi
  fi

  if ! has_openclaw; then
    install_from_pip || true
    promote_binary_to_usr_local || true
  fi

  has_openclaw || {
    echo "Failed to install OpenClaw. Set OPENCLAW_INSTALL_URL or install manually, then re-run verification." >&2
    exit 1
  }

  echo "OpenClaw installed at: $(command -v openclaw)"
}

main "$@"
