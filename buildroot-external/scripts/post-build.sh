#!/bin/bash
# Shared post-build script. Called by Buildroot as:
#   post-build.sh <TARGET_DIR> <BOARD_DIR> <HOOK_FILE>
# The board hook file must define glassos_pre_build().
set -euo pipefail

TARGET_DIR="${1}"
BOARD_DIR="${2}"
HOOK_FILE="${3}"

# BR2_EXTERNAL_GLASSOS_PATH is normally set by Buildroot; fall back to the
# directory containing this script's parent.
BR2_EXTERNAL_GLASSOS_PATH="${BR2_EXTERNAL_GLASSOS_PATH:-$(cd "$(dirname "$(readlink -f "$0")")/.." && pwd)}"

# Load project-level identity.
# shellcheck source=/dev/null
. "${BR2_EXTERNAL_GLASSOS_PATH}/meta"
# Load board identity.
# shellcheck source=/dev/null
. "${BOARD_DIR}/meta"
# shellcheck source=/dev/null
. "${HOOK_FILE}"

# ── RAUC ────────────────────────────────────────────────────────────────────
mkdir -p "${TARGET_DIR}/etc/rauc"

# Render the system.conf template using the board's GLASSOS_COMPATIBLE value.
GLASSOS_COMPATIBLE="${GLASSOS_COMPATIBLE}" \
    envsubst '$GLASSOS_COMPATIBLE' \
    < "${BR2_EXTERNAL_GLASSOS_PATH}/ota/system.conf.tmpl" \
    > "${TARGET_DIR}/etc/rauc/system.conf"

# Install the dev CA as the RAUC keyring.
cp "${BR2_EXTERNAL_GLASSOS_PATH}/ota/dev-ca.pem" \
    "${TARGET_DIR}/etc/rauc/keyring.pem"

# ── Mount points ────────────────────────────────────────────────────────────
mkdir -p "${TARGET_DIR}/boot"
mkdir -p "${TARGET_DIR}/data"
mkdir -p "${TARGET_DIR}/overlay"

# ── Systemd preset ──────────────────────────────────────────────────────────
# Apply service presets so our units are enabled in the target image.
# HOST_DIR is set by Buildroot; systemctl is built as a host tool when
# BR2_INIT_SYSTEMD=y.
"${HOST_DIR}/bin/systemctl" --root="${TARGET_DIR}" preset-all || true

# ── Board hook ──────────────────────────────────────────────────────────────
glassos_pre_build

# ── Permissions ─────────────────────────────────────────────────────────────
chmod +x "${TARGET_DIR}/usr/libexec/glassos-expand"
chmod +x "${TARGET_DIR}/usr/libexec/glassos-wifi-provision"
