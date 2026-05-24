# Adding a Board

This guide describes how to add a new board to GlassOS.

## 1. Create a board directory

```
buildroot-external/board/<board-id>/
```

## 2. Add a `meta` file

The `meta` file provides the board identity consumed by the shared build scripts:

```sh
BOARD_ID=<board-id>
BOARD_NAME="Human Readable Name"
BOOTLOADER=uboot          # or grub
GLASSOS_COMPATIBLE=glassos-<board-id>
```

## 3. Add boot files

At minimum:

- `cmdline.txt` — kernel command line.  Use `root=LABEL=glassos-system0`.
- `config.txt` — RPi firmware config (RPi boards only).
- `linux.config` — board-specific kernel config fragment.
  Common settings (WiFi, systemd) are already in `kernel/common.config`.

## 4. Add a `genimage.cfg`

Define the four standard partitions using the canonical labels:

| Label              | FS    | Size   |
|--------------------|-------|--------|
| `glassos-boot`     | FAT32 | 256 MB |
| `glassos-system0`  | ext4  | ≥512 MB |
| `glassos-system1`  | ext4  | same as system0 |
| `glassos-data`     | ext4  | ≥256 MB |

## 5. Add a `glassos-hook.sh`

Define the two hook functions called by the shared build scripts:

```bash
function glassos_pre_build() {
    # copy board-specific files to BINARIES_DIR, etc.
}

function glassos_post_image() {
    : # compress, convert, etc.
}
```

## 6. Add a defconfig

Copy an existing defconfig and adjust:

- `BR2_LINUX_KERNEL_DEFCONFIG` — upstream defconfig name.
- `BR2_TARGET_UBOOT_BOARD_DEFCONFIG` — U-Boot board config.
- `BR2_LINUX_KERNEL_CONFIG_FRAGMENT_FILES` — include `kernel/common.config` first.
- `BR2_ROOTFS_POST_BUILD_SCRIPT` / `BR2_ROOTFS_POST_IMAGE_SCRIPT` — point to your board shims.

## 7. Add thin post-build/post-image shims

```bash
#!/bin/bash
BOARD_DIR="$(dirname "$(readlink -f "$0")")"
BR2_EXTERNAL_GLASSOS_PATH="$(cd "${BOARD_DIR}/../.." && pwd)"
exec "${BR2_EXTERNAL_GLASSOS_PATH}/scripts/post-build.sh" \
    "$1" "${BOARD_DIR}" "${BOARD_DIR}/glassos-hook.sh"
```

## 8. Add the board to the Makefile

```makefile
BOARDS := rpi4 rpi5 <board-id>
```

