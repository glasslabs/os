#!/bin/bash
# Shared post-image script. Called by Buildroot as:
#   post-image.sh <BINARIES_DIR> <BOARD_DIR> <HOOK_FILE>
# TARGET_DIR is supplied by Buildroot as an environment variable.
# The board hook file must define glassos_pre_image() and glassos_post_image().
set -euo pipefail

BOARD_DIR="${2}"
HOOK_FILE="${3}"

# Load board identity (DTB_GLOB, GLASSOS_COMPATIBLE, …).
# shellcheck source=/dev/null
. "${BOARD_DIR}/meta"
# shellcheck source=/dev/null
. "${HOOK_FILE}"

GENIMAGE_TMP="${BUILD_DIR}/genimage.tmp"
GENIMAGE_CFG="${BOARD_DIR}/genimage.cfg"

# Create a clean boot staging directory.  The board hook populates it with
# every file that should appear in the VFAT boot partition.
BOOT_DATA="${BINARIES_DIR}/boot"
export BOOT_DATA
rm -rf "${BOOT_DATA}"
mkdir -p "${BOOT_DATA}"

# Let the board hook stage all boot files.
glassos_pre_image

rm -rf "${GENIMAGE_TMP}"

# Build the boot VFAT and the full disk image.
# --rootpath points genimage to the pre-staged boot directory so the VFAT is
# populated from its contents rather than from an explicit files list.
genimage \
    --rootpath "${BOOT_DATA}" \
    --tmppath  "${GENIMAGE_TMP}" \
    --inputpath "${BINARIES_DIR}" \
    --outputpath "${BINARIES_DIR}" \
    --config "${GENIMAGE_CFG}"

glassos_post_image
