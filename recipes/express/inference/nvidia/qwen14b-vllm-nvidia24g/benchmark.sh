#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:8000}"
curl -fsS "$BASE_URL/v1/models" >/dev/null

