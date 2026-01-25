#!/usr/bin/env bash
set -euo pipefail

# Remove potential distro-managed ollama packages (best-effort)
if command -v apt-get >/dev/null 2>&1; then
  apt-get remove -y ollama 2>/dev/null || true
fi

if command -v snap >/dev/null 2>&1; then
  snap remove ollama 2>/dev/null || true
fi

# Remove old unit/binary (non-destructive)
bash "$(dirname "$0")/uninstall.sh" || true

echo "Soft cleanup completed (data preserved)."
