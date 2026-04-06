#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/up.sh"

set -a
source "${ENV_FILE}"
set +a

exec "${GO:-go}" test -v -count=1 ./test/...
