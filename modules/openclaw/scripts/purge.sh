#!/usr/bin/env bash
set -euo pipefail

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

bash "$(dirname "$0")/uninstall.sh"

rm -rf "${HOME}/.config/openclaw" || true
rm -rf "${HOME}/.local/share/openclaw" || true
$SUDO rm -rf /var/lib/openclaw || true
