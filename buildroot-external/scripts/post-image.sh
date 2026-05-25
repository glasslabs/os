#!/bin/bash
# Shared post-image script. Called by Buildroot as:
#   post-image.sh <BINARIES_DIR> <BOARD_DIR> <HOOK_FILE>
# TARGET_DIR is supplied by Buildroot as an environment variable.
# The board hook file must define glassos_pre_image() and glassos_post_image().
set -euo pipefail

BOARD_DIR="${2}"
HOOK_FILE="${3}"

# Load project-level identity (GLASSOS_ID, GLASSOS_NAME, version components).
# shellcheck source=/dev/null
. "${BR2_EXTERNAL_GLASSOS_PATH}/meta"
# Load board identity (DTB_GLOB, GLASSOS_COMPATIBLE, partition sizes, …).
# shellcheck source=/dev/null
. "${BOARD_DIR}/meta"
# shellcheck source=/dev/null
. "${HOOK_FILE}"

GENIMAGE_TMP="${BUILD_DIR}/genimage.tmp"

# ── Version ──────────────────────────────────────────────────────────────────
# Can be overridden by passing GLASSOS_VERSION=x.y on the make command line.
GLASSOS_VERSION="${GLASSOS_VERSION:-${GLASSOS_VERSION_MAJOR}.${GLASSOS_VERSION_MINOR}.${GLASSOS_VERSION_PATCH}}"
GLASSOS_IMAGE_BASENAME="${GLASSOS_ID}-${BOARD_ID}-${GLASSOS_VERSION}"

# ── RAUC bundle signing ───────────────────────────────────────────────────────
# Allow CI to inject a production key/cert via environment variables; fall back
# to the committed dev key so local builds always work out of the box.
GLASSOS_RAUC_KEY="${GLASSOS_RAUC_KEY:-${BR2_EXTERNAL_GLASSOS_PATH}/ota/dev-ca.key.pem}"
GLASSOS_RAUC_CERT="${GLASSOS_RAUC_CERT:-${BR2_EXTERNAL_GLASSOS_PATH}/ota/dev-ca.pem}"

# Render the RAUC manifest from the template (envsubst replaces
# $GLASSOS_COMPATIBLE and $GLASSOS_VERSION).
RAUC_MANIFEST=$(GLASSOS_COMPATIBLE="${GLASSOS_COMPATIBLE}" \
    GLASSOS_VERSION="${GLASSOS_VERSION}" \
    envsubst '$GLASSOS_COMPATIBLE $GLASSOS_VERSION' \
    < "${BR2_EXTERNAL_GLASSOS_PATH}/ota/manifest.raucm.tmpl")

# ── Export all variables consumed by genimage config fragments ────────────────
export PARTITION_TABLE_TYPE
export BOOT_SIZE KERNEL_SIZE SYSTEM_SIZE BOOTSTATE_SIZE OVERLAY_SIZE DATA_SIZE
export KERNEL_FILE BINARIES_DIR
export GLASSOS_IMAGE_BASENAME
export GLASSOS_RAUC_KEY GLASSOS_RAUC_CERT RAUC_MANIFEST

# Create a clean boot staging directory.  The board hook populates it with
# every file that should appear in the VFAT boot partition.
BOOT_DATA="${BINARIES_DIR}/boot"
export BOOT_DATA
rm -rf "${BOOT_DATA}"
mkdir -p "${BOOT_DATA}"

# Let the board hook stage all boot files.
glassos_pre_image

rm -rf "${GENIMAGE_TMP}"

# Build all intermediate images (boot.vfat, kernel.img, overlay.ext4, data.ext4)
# and the full disk image in a single genimage pass.
# --rootpath points genimage to the pre-staged boot directory so the VFAT is
# populated from its contents rather than from an explicit files list.
# --includepath searches the board directory first (for any board-specific
# overrides) then the shared genimage fragment directory.
genimage \
    --rootpath "${BOOT_DATA}" \
    --tmppath  "${GENIMAGE_TMP}" \
    --inputpath "${BINARIES_DIR}" \
    --outputpath "${BINARIES_DIR}" \
    --includepath "${BOARD_DIR}:${BR2_EXTERNAL_GLASSOS_PATH}/genimage"

glassos_post_image
