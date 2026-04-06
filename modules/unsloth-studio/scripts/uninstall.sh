#!/usr/bin/env bash
set -euo pipefail

studio_home="${UNSLOTH_STUDIO_HOME:-${HOME}/.unsloth/studio}"
share_dir="${HOME}/.local/share/unsloth"

rm -rf "$studio_home"

if [[ -f "${share_dir}/studio.conf" ]] && grep -Fq "$studio_home" "${share_dir}/studio.conf"; then
  rm -f "${share_dir}/studio.conf" "${share_dir}/launch-studio.sh"
fi
