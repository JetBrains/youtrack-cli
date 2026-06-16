#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

copy_logs
wait_for_auth 180 2

require_command jq

services_json="$(hub_api_get "/services?fields=id,name,applicationName")"
youtrack_service_id="$(printf '%s' "${services_json}" | jq -r '.services[] | select(.applicationName=="YouTrack") | .id')"

if [[ -z "${youtrack_service_id}" ]]; then
  echo "failed to resolve YouTrack service id from Hub response" >&2
  echo "${services_json}" >&2
  exit 1
fi

token_response="$(
  hub_api_post \
    "/users/me/permanenttokens?fields=id,token" \
    "$(cat <<EOF
{"name":"youtrack-cli-e2e","scope":[{"id":"0-0-0-0-0"},{"id":"${youtrack_service_id}"}]}
EOF
)"
)"

token="$(printf '%s' "${token_response}" | jq -r '.token')"

if [[ -z "${token}" || "${token}" == "null" ]]; then
  echo "failed to create permanent token" >&2
  echo "${token_response}" >&2
  exit 1
fi

write_env_file "${token}"

if ! token_is_valid "${token}"; then
  echo "generated token is not accepted by YouTrack" >&2
  exit 1
fi

copy_logs
echo "prepared YouTrack E2E instance at ${BASE_URL}"

# --- Gitea setup ---
GITEA_HOST="${GITEA_HOST:-localhost}"
GITEA_PORT="${GITEA_E2E_PORT:-3000}"
GITEA_URL="http://${GITEA_HOST}:${GITEA_PORT}"
GITEA_ADMIN_USER="e2eadmin"
GITEA_ADMIN_PASS="e2epassword"
GITEA_ORG="youtrack-cli-e2e"

echo "Setting up Gitea..."
if ! curl -sf "${GITEA_URL}/api/v1/version" -o /dev/null; then
  echo "Gitea is not reachable at ${GITEA_URL} — skipping Gitea setup" >&2
  # Write placeholder values so tests skip gracefully
  grep -v "^GITEA_BASE_URL=" "${ENV_FILE}" > "${ENV_FILE}.tmp" && mv "${ENV_FILE}.tmp" "${ENV_FILE}" || true
  echo "GITEA_BASE_URL=" >> "${ENV_FILE}"
  grep -v "^GITEA_TOKEN=" "${ENV_FILE}" > "${ENV_FILE}.tmp" && mv "${ENV_FILE}.tmp" "${ENV_FILE}" || true
  echo "GITEA_TOKEN=" >> "${ENV_FILE}"
else
  # Create admin user (idempotent — 422 if already exists)
  curl -sf -X POST "${GITEA_URL}/api/v1/users/search" -o /dev/null || true
  curl -sf -X POST "${GITEA_URL}/api/v1/admin/users" \
    -H "Content-Type: application/json" \
    --user "${GITEA_ADMIN_USER}:${GITEA_ADMIN_PASS}" \
    -d "{\"login_name\":\"${GITEA_ADMIN_USER}\",\"username\":\"${GITEA_ADMIN_USER}\",\"email\":\"admin@example.com\",\"password\":\"${GITEA_ADMIN_PASS}\",\"must_change_password\":false}" \
    -o /dev/null 2>/dev/null || true

  # Create org + repo
  curl -sf -X POST "${GITEA_URL}/api/v1/orgs" \
    --user "${GITEA_ADMIN_USER}:${GITEA_ADMIN_PASS}" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${GITEA_ORG}\",\"visibility\":\"public\"}" -o /dev/null || true
  curl -sf -X POST "${GITEA_URL}/api/v1/orgs/${GITEA_ORG}/repos" \
    --user "${GITEA_ADMIN_USER}:${GITEA_ADMIN_PASS}" \
    -H "Content-Type: application/json" \
    -d '{"name":"test-repo","auto_init":true}' -o /dev/null || true

  # Export GITEA_BASE_URL (idempotent)
  grep -v "^GITEA_BASE_URL=" "${ENV_FILE}" > "${ENV_FILE}.tmp" && mv "${ENV_FILE}.tmp" "${ENV_FILE}" || true
  echo "GITEA_BASE_URL=${GITEA_URL}" >> "${ENV_FILE}"

  # Generate token (idempotent — delete existing first, then create)
  grep -v "^GITEA_TOKEN=" "${ENV_FILE}" > "${ENV_FILE}.tmp" && mv "${ENV_FILE}.tmp" "${ENV_FILE}" || true
  # Delete existing token if present (idempotent — ignore 404)
  curl -sf -X DELETE "${GITEA_URL}/api/v1/users/${GITEA_ADMIN_USER}/tokens/e2e-token" \
    --user "${GITEA_ADMIN_USER}:${GITEA_ADMIN_PASS}" -o /dev/null 2>/dev/null || true

  GITEA_TOKEN_RESPONSE="$(curl -sf -X POST "${GITEA_URL}/api/v1/users/${GITEA_ADMIN_USER}/tokens" \
    --user "${GITEA_ADMIN_USER}:${GITEA_ADMIN_PASS}" \
    -H "Content-Type: application/json" \
    -d '{"name":"e2e-token"}')"
  GITEA_TOKEN="$(printf '%s' "${GITEA_TOKEN_RESPONSE}" | jq -r '.sha1 // empty')"
  if [[ -z "${GITEA_TOKEN}" ]]; then
    echo "failed to create Gitea token; response: ${GITEA_TOKEN_RESPONSE}" >&2
    exit 1
  fi
  echo "GITEA_TOKEN=${GITEA_TOKEN}" >> "${ENV_FILE}"
fi

# --- MailHog setup ---
MAILHOG_HOST="${MAILHOG_HOST:-localhost}"
MAILHOG_PORT="${MAILHOG_SMTP_PORT:-1025}"
grep -v "^MAILHOG_HOST=\|^MAILHOG_PORT=" "${ENV_FILE}" > "${ENV_FILE}.tmp" && mv "${ENV_FILE}.tmp" "${ENV_FILE}" || true
echo "MAILHOG_HOST=${MAILHOG_HOST}" >> "${ENV_FILE}"
echo "MAILHOG_PORT=${MAILHOG_PORT}" >> "${ENV_FILE}"
echo "Setup complete: Gitea at ${GITEA_URL}, MailHog SMTP at ${MAILHOG_HOST}:${MAILHOG_PORT}"
