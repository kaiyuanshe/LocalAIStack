#!/usr/bin/env sh
set -eu

BASE_URL="${BASE_URL:-http://127.0.0.1:18080}"
curl -fsS "$BASE_URL/health" >/dev/null

