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
