#!/usr/bin/env bash
set -euo pipefail

systemctl stop ollama 2>/dev/null || true

# Best-effort remove unit file if present
rm -f /etc/systemd/system/ollama.service
rm -f /lib/systemd/system/ollama.service

# Remove binary if installed by the upstream installer
rm -f /usr/local/bin/ollama

systemctl daemon-reload || true
echo "Rollback completed (user data preserved)."
