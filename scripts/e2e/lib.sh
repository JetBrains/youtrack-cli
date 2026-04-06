#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
E2E_DIR="${ROOT_DIR}/.e2e"
BASE_URL="http://127.0.0.1:8088"
YOUTRACK_E2E_VERSION="${YOUTRACK_E2E_VERSION:-2026.1.12024}"
HUB_API_URL="${BASE_URL}/hub/api/rest"
ENV_FILE="${E2E_DIR}/youtrack.env"
LOG_DIR="${E2E_DIR}/logs"

ADMIN_LOGIN="admin"
ADMIN_PASSWORD="admin"
LICENSE_KEY="feefab08da9927c9ee29cf7463f228307221fbf5c5e47336e9df87341c2baaabcfbda7635d8d5925a416af13d47741b5d22f1442d76853c54312574c03a2bf783845e7d20929b9a7961b6475e68bc59134ed872ceae3a7fa17d4c17a79552b19afed48f8dc0f6f920a3a23f898772671845a9392576a6b392bd38c1b4ba73dca"

export YOUTRACK_E2E_VERSION

compose() {
  docker compose -p "youtrack-cli-e2e" -f "${ROOT_DIR}/docker-compose.e2e.yml" "$@"
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

ensure_dirs() {
  mkdir -p "${E2E_DIR}/data" "${E2E_DIR}/conf" "${LOG_DIR}" "${E2E_DIR}/backups"
  chmod -R 777 "${E2E_DIR}/data" "${E2E_DIR}/conf" "${LOG_DIR}" "${E2E_DIR}/backups"
}

reset_state() {
  rm -f "${ENV_FILE}"
  rm -rf "${E2E_DIR}/data" "${E2E_DIR}/conf" "${LOG_DIR}" "${E2E_DIR}/backups"
  ensure_dirs
}

write_env_file() {
  local token="$1"

  cat >"${ENV_FILE}" <<EOF
YOUTRACK_BASE_URL=${BASE_URL}
YOUTRACK_TOKEN=${token}
EOF
}

wait_for_auth() {
  local attempts="${1:-180}"
  local sleep_seconds="${2:-2}"
  local response

  for _ in $(seq 1 "${attempts}"); do
    response="$(
      curl -fsS \
        -u "${ADMIN_LOGIN}:${ADMIN_PASSWORD}" \
        -H "Accept: application/json" \
        "${HUB_API_URL}/users/me?fields=id,login" \
        || true
    )"
    if printf '%s' "${response}" | grep -q '"login"'; then
      return 0
    fi
    sleep "${sleep_seconds}"
  done

  echo "timed out waiting for authenticated Hub API at ${HUB_API_URL}" >&2
  return 1
}

copy_logs() {
  mkdir -p "${LOG_DIR}"
  compose logs --no-color youtrack >"${LOG_DIR}/youtrack.log" 2>&1 || true
}

api_with_token() {
  local token="$1"
  local path="$2"

  curl -fsS \
    -H "Authorization: Bearer ${token}" \
    -H "Accept: application/json" \
    "${BASE_URL}${path}"
}

current_token() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    return 1
  fi

  awk -F= '$1 == "YOUTRACK_TOKEN" { print substr($0, index($0, "=") + 1) }' "${ENV_FILE}"
}

token_is_valid() {
  local token="$1"
  api_with_token "${token}" "/api/users/me?fields=id,login" >/dev/null
}

service_is_running() {
  compose ps --status running -q youtrack | grep -q .
}

configure_instance() {
  compose run --rm --no-deps youtrack \
    configure \
    "--base-url=${BASE_URL}" \
    "--listen-port=8080" \
    "--disable.configurationWizard=true" \
    "-J-Ddisable.configuration.wizard.on.clean.install=true" \
    "-J-Ddisable.configuration.wizard.on.upgrade=true" \
    "-J-Djetbrains.youtrack.licenseName=e2e-tests" \
    "-J-Djetbrains.youtrack.licenseKey=${LICENSE_KEY}" \
    "--root-user=${ADMIN_LOGIN}" \
    "--root-password=${ADMIN_PASSWORD}"
}

hub_api_get() {
  local path="$1"

  curl -fsS \
    -u "${ADMIN_LOGIN}:${ADMIN_PASSWORD}" \
    -H "Accept: application/json" \
    "${HUB_API_URL}${path}"
}

hub_api_post() {
  local path="$1"
  local body="$2"

  curl -fsS \
    -u "${ADMIN_LOGIN}:${ADMIN_PASSWORD}" \
    -H "Accept: application/json" \
    -H "Content-Type: application/json" \
    -X POST \
    "${HUB_API_URL}${path}" \
    -d "${body}"
}
