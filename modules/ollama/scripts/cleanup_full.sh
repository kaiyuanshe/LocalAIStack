#!/usr/bin/env bash
set -euo pipefail

bash "$(dirname "$0")/purge.sh"
echo "Full cleanup completed (destructive)."
