#!/bin/bash
# Thin shim — delegates to the shared post-image script.
set -euo pipefail

BOARD_DIR="$(dirname "$(readlink -f "$0")")"
BR2_EXTERNAL_GLASSOS_PATH="$(cd "${BOARD_DIR}/../.." && pwd)"

exec "${BR2_EXTERNAL_GLASSOS_PATH}/scripts/post-image.sh" \
    "$1" \
    "${BOARD_DIR}" \
    "${BOARD_DIR}/glassos-hook.sh"

