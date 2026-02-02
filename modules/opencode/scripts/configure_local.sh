#!/usr/bin/env bash
set -euo pipefail

config_dir="${HOME}/.config/opencode"
config_path="${config_dir}/opencode.json"

mkdir -p "$config_dir"

python3 - <<'PY'
import json
import os
import re
import shutil
import subprocess

config_path = os.environ.get("OPENCODE_CONFIG_PATH", os.path.join(os.path.expanduser("~"), ".config", "opencode", "opencode.json"))

def load_config(path):
    if not os.path.exists(path):
        return {}
    try:
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception as exc:
        print(f"OpenCode config is not valid JSON; skipping auto-config: {exc}")
        return None

def ensure_provider(config, key, name, base_url):
    providers = config.setdefault("provider", {})
    entry = providers.setdefault(key, {})
    entry.setdefault("name", name)
    entry.setdefault("npm", "@ai-sdk/openai-compatible")
    options = entry.setdefault("options", {})
    options.setdefault("baseURL", base_url)
    entry.setdefault("models", {})
    return entry

def add_models(entry, models):
    existing = entry.setdefault("models", {})
    for model_id, model_name in models:
        if model_id not in existing:
            existing[model_id] = {"name": model_name}

def find_local_gguf_models():
    models_dir = os.path.join(os.path.expanduser("~"), ".localaistack", "models")
    results = []
    if not os.path.isdir(models_dir):
        return results
    for root, _, files in os.walk(models_dir):
        for filename in files:
            if filename.lower().endswith(".gguf"):
                model_id = os.path.splitext(filename)[0]
                results.append((model_id, model_id))
    return results

def parse_ollama_list():
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
        if not parts:
            continue
        name = parts[0]
        models.append((name, name))
    return models

config = load_config(config_path)
if config is None:
    raise SystemExit(0)

config.setdefault("$schema", "https://opencode.ai/config.json")

llama_entry = ensure_provider(config, "llama.cpp", "llama.cpp (local)", "http://127.0.0.1:8080/v1")
add_models(llama_entry, find_local_gguf_models())

ollama_entry = ensure_provider(config, "ollama", "Ollama (local)", "http://localhost:11434/v1")
add_models(ollama_entry, parse_ollama_list())

ensure_provider(config, "lmstudio", "LM Studio (local)", "http://127.0.0.1:1234/v1")

with open(config_path, "w", encoding="utf-8") as f:
    json.dump(config, f, indent=2, ensure_ascii=True)
    f.write("\n")

print(f"OpenCode local providers configured in {config_path}")
PY
