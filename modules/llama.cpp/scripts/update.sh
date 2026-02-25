#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Keep the previously used installation strategy:
# - source install leaves a git repo in /usr/local/llama.cpp
# - otherwise fallback to binary upgrade
if [[ -d /usr/local/llama.cpp/.git ]]; then
  echo "Detected source-based llama.cpp installation; updating from source."
  bash "$script_dir/install_source.sh"
  exit 0
fi

if [[ -x /usr/local/bin/llama-server || -x /usr/local/bin/llama-cli ]]; then
  echo "Detected binary-based llama.cpp installation; updating via latest release binaries."
  bash "$script_dir/install_binary.sh"
  exit 0
fi

echo "No existing llama.cpp installation detected; installing via binaries."
bash "$script_dir/install_binary.sh"
