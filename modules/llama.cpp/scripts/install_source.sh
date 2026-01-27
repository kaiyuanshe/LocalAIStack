#!/usr/bin/env bash
set -euo pipefail

repo_url="https://github.com/ggerganov/llama.cpp.git"
source_dir="/usr/local/llama.cpp"

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

if [[ -d "$source_dir/.git" ]]; then
  $SUDO git -C "$source_dir" fetch --tags --prune
  $SUDO git -C "$source_dir" reset --hard origin/master
else
  $SUDO rm -rf "$source_dir"
  $SUDO git clone --depth 1 "$repo_url" "$source_dir"
fi

$SUDO cmake -S "$source_dir" -B "$source_dir/build" -DLLAMA_BUILD_SERVER=ON -DCMAKE_BUILD_TYPE=Release
$SUDO cmake --build "$source_dir/build" --config Release

for bin in llama-cli llama-server; do
  if [[ -x "$source_dir/build/bin/$bin" ]]; then
    $SUDO install -m 0755 "$source_dir/build/bin/$bin" "/usr/local/bin/$bin"
  fi
done

if [[ ! -x /usr/local/bin/llama-cli ]]; then
  echo "llama-cli was not built successfully." >&2
  exit 1
fi
