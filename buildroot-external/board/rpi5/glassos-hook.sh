#!/bin/bash
# Raspberry Pi 5 board hook for GlassOS.
# Sourced by buildroot-external/scripts/post-build.sh and post-image.sh.

function glassos_pre_build() {
    # Copy board-specific boot files into BINARIES_DIR for genimage.
    cp -v "${BOARD_DIR}/config.txt"  "${BINARIES_DIR}/"
    cp -v "${BOARD_DIR}/cmdline.txt" "${BINARIES_DIR}/"
}

function glassos_post_image() {
    : # nothing extra needed for rpi5
}

