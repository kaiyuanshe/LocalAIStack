#!/usr/bin/env bash
set -euo pipefail

install_dir="/usr/local/llama.cpp"

if command -v sudo >/dev/null 2>&1 && [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  SUDO="sudo"
else
  SUDO=""
fi

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64)
    arch="x86_64"
    patterns=("bin-ubuntu-x64.tar.gz" "bin-ubuntu-vulkan-x64.tar.gz" "bin-openEuler-x86.tar.gz")
    ;;
  aarch64|arm64)
    arch="aarch64"
    patterns=("bin-310p-openEuler-aarch64.tar.gz" "bin-openEuler-aarch64.tar.gz")
    ;;
  *)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset_json=$(python3 - <<'PY'
import json,urllib.request
url='https://api.github.com/repos/ggerganov/llama.cpp/releases/latest'
with urllib.request.urlopen(url, timeout=30) as r:
    data=json.load(r)
print(json.dumps(data.get('assets', [])))
PY
)

patterns_json=$(printf '%s\n' "${patterns[@]}" | python3 - <<'PY'
import json,sys
patterns=[line.strip() for line in sys.stdin if line.strip()]
print(json.dumps(patterns))
PY
)

asset_url=$(ASSET_JSON="$asset_json" PATTERNS_JSON="$patterns_json" python3 - <<'PY'
import json,os
assets=json.loads(os.environ.get("ASSET_JSON","[]"))
patterns=json.loads(os.environ.get("PATTERNS_JSON","[]"))
for pattern in patterns:
    for asset in assets:
        name=asset.get('name','')
        if pattern in name:
            print(asset.get('browser_download_url',''))
            raise SystemExit(0)
print('')
PY
)

if [[ -z "$asset_url" ]]; then
  echo "No suitable prebuilt asset found for $arch." >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

archive="$tmp_dir/llama.cpp.tar.gz"
curl -fsSL "$asset_url" -o "$archive"

$SUDO rm -rf "$install_dir"
$SUDO mkdir -p "$install_dir"
$SUDO tar -xzf "$archive" -C "$install_dir"

for bin in llama-cli llama-server; do
  bin_path=$(find "$install_dir" -type f -name "$bin" -perm -111 | head -n 1 || true)
  if [[ -n "$bin_path" ]]; then
    $SUDO install -m 0755 "$bin_path" "/usr/local/bin/$bin"
  fi
done

if [[ ! -x /usr/local/bin/llama-cli ]]; then
  echo "llama-cli was not installed. Check extracted archive contents." >&2
  exit 1
fi
