#!/bin/sh
set -e

BOARD_DIR="$(dirname "$(readlink -f "$0")")"
BINARIES_DIR="${BINARIES_DIR}"
BUILD_DIR="${BUILD_DIR}"

GENIMAGE_TMP="${BUILD_DIR}/genimage.tmp"
GENIMAGE_CFG="${BOARD_DIR}/genimage.cfg"

rm -rf "${GENIMAGE_TMP}"

genimage \
    --rootpath "${TARGET_DIR}" \
    --tmppath  "${GENIMAGE_TMP}" \
    --inputpath "${BINARIES_DIR}" \
    --outputpath "${BINARIES_DIR}" \
    --config "${GENIMAGE_CFG}"

