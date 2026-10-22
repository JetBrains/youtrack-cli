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
fi
compose up -d --wait --wait-timeout 600 youtrack gitea mailhog

if token="$(current_token 2>/dev/null)" && token_is_valid "${token}"; then
  apply_license "${token}"
  echo "reusing existing YouTrack E2E instance at ${BASE_URL}"
  # Patch stale env files that predate YOUTRACK_LOGIN/PASSWORD being written.
  if ! grep -q "^YOUTRACK_LOGIN=" "${ENV_FILE}" 2>/dev/null; then
    echo "YOUTRACK_LOGIN=${ADMIN_LOGIN}" >> "${ENV_FILE}"
    echo "YOUTRACK_PASSWORD=${ADMIN_PASSWORD}" >> "${ENV_FILE}"
  fi
  exit 0
fi

"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/bootstrap.sh"
