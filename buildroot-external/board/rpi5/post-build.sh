#!/bin/sh
set -e

BOARD_DIR="$(dirname "$(readlink -f "$0")")"
BINARIES_DIR="${BINARIES_DIR}"
TARGET_DIR="${TARGET_DIR}"

# Copy board-specific boot files into BINARIES_DIR for genimage.
cp -v "${BOARD_DIR}/config.txt"  "${BINARIES_DIR}/"
cp -v "${BOARD_DIR}/cmdline.txt" "${BINARIES_DIR}/"

# Create /boot and /data mount points in the rootfs.
mkdir -p "${TARGET_DIR}/boot"
mkdir -p "${TARGET_DIR}/data"

# Add /data and /boot to fstab.
cat >> "${TARGET_DIR}/etc/fstab" <<'EOF'
/dev/mmcblk0p1  /boot  vfat  defaults,ro  0  2
/dev/mmcblk0p4  /data  ext4  defaults     0  2
EOF

