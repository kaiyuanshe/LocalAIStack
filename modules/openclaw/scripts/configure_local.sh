#!/usr/bin/env bash
set -euo pipefail

config_dir="${HOME}/.config/openclaw"
config_path="${config_dir}/config.json"

mkdir -p "$config_dir"

OPENCLAW_CONFIG_PATH="$config_path" python3 - <<'PY'
import json
import os
import re
import shutil
import subprocess

config_path = os.environ.get("OPENCLAW_CONFIG_PATH", os.path.join(os.path.expanduser("~"), ".config", "openclaw", "config.json"))


def load_config(path):
    if not os.path.exists(path):
        return {}
    try:
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception as exc:
        print(f"OpenClaw config is not valid JSON; skipping auto-config: {exc}")
        return None


def ensure_provider(config, key, provider_type, base_url):
    providers = config.setdefault("providers", {})
    provider = providers.setdefault(key, {})
    provider.setdefault("type", provider_type)
    provider.setdefault("base_url", base_url)
    provider.setdefault("models", [])
    return provider


def parse_ollama_models():
    if shutil.which("ollama") is None:
        return []
    try:
        output = subprocess.check_output(["ollama", "list"], text=True)
    except Exception:
        return []

    lines = [line.strip() for line in output.splitlines() if line.strip()]
    if len(lines) <= 1:
        return []

    models = []
    for line in lines[1:]:
        parts = re.split(r"\s+", line)
        if parts:
            models.append(parts[0])
    return models


def parse_local_gguf_models():
    models_root = os.path.join(os.path.expanduser("~"), ".localaistack", "models")
    found = []
    if not os.path.isdir(models_root):
        return found

    for root, _, files in os.walk(models_root):
        for file_name in files:
            if file_name.lower().endswith(".gguf"):
                found.append(os.path.splitext(file_name)[0])
    return sorted(set(found))


cfg = load_config(config_path)
if cfg is None:
    raise SystemExit(0)

cfg.setdefault("schema", "https://openclaw.ai/config.schema.json")

ollama_provider = ensure_provider(cfg, "ollama", "openai-compatible", "http://127.0.0.1:11434/v1")
for model in parse_ollama_models():
    if model not in ollama_provider["models"]:
        ollama_provider["models"].append(model)

llama_cpp_provider = ensure_provider(cfg, "llama.cpp", "openai-compatible", "http://127.0.0.1:8080/v1")
for model in parse_local_gguf_models():
    if model not in llama_cpp_provider["models"]:
        llama_cpp_provider["models"].append(model)

with open(config_path, "w", encoding="utf-8") as f:
    json.dump(cfg, f, indent=2, ensure_ascii=False)
    f.write("\n")

print(f"OpenClaw local providers configured in {config_path}")
PY
