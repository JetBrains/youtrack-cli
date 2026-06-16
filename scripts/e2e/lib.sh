#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
E2E_DIR="${ROOT_DIR}/.e2e"
ENV_FILE="${E2E_DIR}/youtrack.env"
PORT_FILE="${E2E_DIR}/port"
LOG_DIR="${E2E_DIR}/logs"
YOUTRACK_E2E_VERSION="${YOUTRACK_E2E_VERSION:-2026.1.12024}"

find_free_port() {
  local port
  for port in $(seq 8080 8099); do
    if ! lsof -i ":${port}" -sTCP:LISTEN >/dev/null 2>&1; then
      echo "${port}"
      return 0
    fi
  done
  echo "no free port found in range 8080-8099" >&2
  return 1
}

if [[ -z "${YOUTRACK_E2E_PORT:-}" ]]; then
  if [[ -f "${PORT_FILE}" ]]; then
    YOUTRACK_E2E_PORT="$(cat "${PORT_FILE}")"
  else
    mkdir -p "${E2E_DIR}"
    YOUTRACK_E2E_PORT="$(find_free_port)"
    echo "${YOUTRACK_E2E_PORT}" > "${PORT_FILE}"
  fi
  export YOUTRACK_E2E_PORT
fi

BASE_URL="http://127.0.0.1:${YOUTRACK_E2E_PORT}"
HUB_API_URL="${BASE_URL}/hub/api/rest"

ADMIN_LOGIN="admin"
ADMIN_PASSWORD="admin"

# Free license (100 users, 10 agents, valid till 2054-01-01 UTC).
# Override with YOUTRACK_E2E_LICENSE_NAME / YOUTRACK_E2E_LICENSE_KEY if needed.
DEFAULT_LICENSE_NAME="youtrack-cli-e2e"
DEFAULT_LICENSE_KEY="de12ec0cb21184a6c12fd65e5dc5c56573117f262e57eeab4994b42bf73ca8db699fc4ced4cc4f92180964c073735f254b0d7423c5bd640505d9fa9e9815e84aed04481134212d5fe9671d563d958c3f6fbbbbb40df8d5872fd0978d93af234af34c0de13f2ee6c5412f620d17b719c754bf2ae7699e592290c6e9970153f2ab"

export YOUTRACK_E2E_VERSION

LICENSE_NAME="${YOUTRACK_E2E_LICENSE_NAME:-${DEFAULT_LICENSE_NAME}}"
LICENSE_KEY="${YOUTRACK_E2E_LICENSE_KEY:-${DEFAULT_LICENSE_KEY}}"

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
YOUTRACK_LOGIN=${ADMIN_LOGIN}
YOUTRACK_PASSWORD=${ADMIN_PASSWORD}
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

  curl -fsS --connect-timeout 5 \
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
    "-J-Djetbrains.youtrack.licenseName=${LICENSE_NAME}" \
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
