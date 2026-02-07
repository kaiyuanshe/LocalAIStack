#!/usr/bin/env bash
set -euo pipefail

COMFYUI_HOME="${COMFYUI_HOME:-$HOME/.local/share/localaistack/comfyui}"
COMFYUI_REPO="${COMFYUI_REPO:-https://github.com/comfyanonymous/ComfyUI.git}"
COMFYUI_REF="${COMFYUI_REF:-master}"
PYTHON_BIN="${COMFYUI_PYTHON:-python3}"

mkdir -p "$(dirname "$COMFYUI_HOME")"

if [[ ! -d "$COMFYUI_HOME/.git" ]]; then
  git clone "$COMFYUI_REPO" "$COMFYUI_HOME"
fi

cd "$COMFYUI_HOME"
git fetch --tags --prune origin
git checkout "$COMFYUI_REF"
git pull --ff-only origin "$COMFYUI_REF"

if [[ ! -d ".venv" ]]; then
  "$PYTHON_BIN" -m venv .venv
fi

# shellcheck disable=SC1091
source .venv/bin/activate
python -m pip install --upgrade pip
python -m pip install -r requirements.txt

mkdir -p "$HOME/.local/bin"
cat > "$HOME/.local/bin/comfyui-las" <<'WRAPPER'
#!/usr/bin/env bash
set -euo pipefail
COMFYUI_HOME="${COMFYUI_HOME:-__COMFYUI_HOME__}"
cd "$COMFYUI_HOME"
source .venv/bin/activate
exec python main.py "$@"
WRAPPER
sed -i "s|__COMFYUI_HOME__|$COMFYUI_HOME|g" "$HOME/.local/bin/comfyui-las"
chmod 0755 "$HOME/.local/bin/comfyui-las"

echo "ComfyUI installed at: $COMFYUI_HOME"
echo "Start command: comfyui-las --listen 127.0.0.1 --port 8188"
