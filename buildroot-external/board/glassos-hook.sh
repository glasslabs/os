#!/bin/bash
# Shared GlassOS board hook for Raspberry Pi boards.
# Sourced by scripts/post-build.sh and scripts/post-image.sh.
# Reads DTB_GLOB from the board's meta file (set before this hook is sourced).

function glassos_pre_build() {
    : # All boot file staging happens in glassos_pre_image(); no-op here.
}

function glassos_pre_image() {
    # rpi-firmware installs to BINARIES_DIR/rpi-firmware/ — copy everything out.
    cp -r "${BINARIES_DIR}/rpi-firmware/"* "${BOOT_DATA}/"

    # Overwrite the pre-built firmware DTBs with kernel-built versions so they
    # always match the custom kernel being used.
    # DTB_GLOB comes from the board's meta file (e.g. "bcm2711-*.dtb").
    cp "${BINARIES_DIR}"/${DTB_GLOB} "${BOOT_DATA}/"

    # U-Boot binary and the compiled A/B boot script.
    cp "${BINARIES_DIR}/u-boot.bin" "${BOOT_DATA}/"
    cp "${BINARIES_DIR}/boot.scr"   "${BOOT_DATA}/"

    # Use a board-specific config.txt if present, otherwise use the shared one.
    if [ -f "${BOARD_DIR}/config.txt" ]; then
        cp "${BOARD_DIR}/config.txt"  "${BOOT_DATA}/"
    else
        cp "${BOARD_DIR}/../config.txt" "${BOOT_DATA}/"
    fi

    # Same fallback pattern for cmdline.txt.
    if [ -f "${BOARD_DIR}/cmdline.txt" ]; then
        cp "${BOARD_DIR}/cmdline.txt" "${BOOT_DATA}/"
    else
        cp "${BOARD_DIR}/../cmdline.txt" "${BOOT_DATA}/"
    fi
}

function glassos_post_image() {
    echo "Compressing ${BINARIES_DIR}/sdcard.img ..."
    xz -T0 -f "${BINARIES_DIR}/sdcard.img"
    echo "Compressed image: ${BINARIES_DIR}/sdcard.img.xz"
}
