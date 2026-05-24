#!/bin/bash
# Shared post-image script. Called by Buildroot as:
#   post-image.sh <TARGET_DIR> <BOARD_DIR> <HOOK_FILE>
# The board hook file must define glassos_post_image().
set -euo pipefail

BOARD_DIR="${2}"
HOOK_FILE="${3}"

# shellcheck source=/dev/null
. "${HOOK_FILE}"

GENIMAGE_TMP="${BUILD_DIR}/genimage.tmp"
GENIMAGE_CFG="${BOARD_DIR}/genimage.cfg"

rm -rf "${GENIMAGE_TMP}"

genimage \
    --rootpath "${TARGET_DIR}" \
    --tmppath  "${GENIMAGE_TMP}" \
    --inputpath "${BINARIES_DIR}" \
    --outputpath "${BINARIES_DIR}" \
    --config "${GENIMAGE_CFG}"

glassos_post_image

