#!/usr/bin/env bash
set -euo pipefail

systemctl stop ollama 2>/dev/null || true
systemctl disable ollama 2>/dev/null || true

rm -f /etc/systemd/system/ollama.service
rm -f /lib/systemd/system/ollama.service
rm -f /usr/local/bin/ollama

systemctl daemon-reload || true
echo "Ollama uninstalled (data preserved at ~/.ollama)."
