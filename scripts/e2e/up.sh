#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

require_command docker
require_command curl

ensure_dirs

if [[ "${YOUTRACK_E2E_RESET:-0}" == "1" ]]; then
  compose down -v --remove-orphans || true
  reset_state
fi

if ! service_is_running; then
  configure_instance
  compose up -d --wait --wait-timeout 600 youtrack
fi

if token="$(current_token 2>/dev/null)" && token_is_valid "${token}"; then
  echo "reusing existing YouTrack E2E instance at ${BASE_URL}"
  exit 0
fi

"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/bootstrap.sh"
