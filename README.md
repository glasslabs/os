# GlassOS

Minimal custom Linux OS image for [Looking Glass](https://github.com/glasslabs/looking-glass)
smart mirrors, targeting Raspberry Pi 4 and Pi 5.

Built with [Buildroot](https://buildroot.org). Boots directly into Looking Glass on the
framebuffer via `cage` (a single-application Wayland compositor over DRM/KMS) — no desktop
environment, no X11.

A small Go management agent (`glass-agent`) supervises the `glass` process and hosts an
HTTP API for OTA updates, log access, WiFi configuration, and config/asset/module uploads.

---

## Architecture

```
BusyBox init
  ├── S30data   → mount /data (ext4), create subdirs on first boot
  ├── S40wifi   → migrate wpa_supplicant.conf from /boot, start wpa_supplicant + dhcpcd
  └── respawn: glass-agent  (supervisor + HTTP :8080)
                  └── cage
                        └── glass run ...
```

### Partitions

```
/dev/mmcblk0p1   FAT32   256 MB   /boot    U-Boot + kernel + RPi firmware
/dev/mmcblk0p2   ext4    1.5 GB   /        rootfs slot A (active)
/dev/mmcblk0p3   ext4    1.5 GB   /        rootfs slot B (RAUC target)
/dev/mmcblk0p4   ext4    rest     /data    config, assets, modules (never wiped)
```

---

## Prerequisites

**Host packages (Ubuntu/Debian):**
```sh
sudo apt-get install -y \
  build-essential bc cpio rsync unzip \
  libncurses-dev file wget python3 \
  python3-setuptools libssl-dev bison flex genimage
```

**Clone with submodules:**
```sh
git clone --recurse-submodules https://github.com/glasslabs/os
# or, after cloning:
git submodule update --init
```

---

## Building

```sh
# Pi 4
make build-rpi4

# Pi 5
make build-rpi5
```

The first build downloads sources and compiles the toolchain — allow ~90 minutes.
Subsequent builds with a warm cache take ~5–10 minutes.

Output image: `buildroot/output/<board>/images/sdcard.img`

Before building, set the Looking Glass version in the defconfig:
```
BR2_PACKAGE_GLASS_VERSION="v1.2.3"
```
Or pass it as an override: `make build-rpi4 BR2_PACKAGE_GLASS_VERSION=v1.2.3`

---

## Flashing

```sh
# Flash Pi 4 image to /dev/sdX
make flash BOARD=rpi4 DEV=/dev/sdX

# Flash and write WiFi credentials in one step
make flash BOARD=rpi4 DEV=/dev/sdX SSID="MyNetwork" PSK="mypassword"
```

You can also write WiFi credentials manually before inserting the card:
mount the FAT32 `/boot` partition from your Mac/PC and place a
`wpa_supplicant.conf` file on it (standard Raspberry Pi OS format).
It is migrated to `/data/config/` on first boot and removed from `/boot`.

**Example `wpa_supplicant.conf`:**
```
country=GB
ctrl_interface=DIR=/var/run/wpa_supplicant GROUP=netdev
update_config=1

network={
    ssid="MyNetwork"
    psk="mypassword"
}
```

---

## First Boot

1. Insert SD card, connect display and power.
2. Find the device IP via your router's DHCP table (hostname: `glass`).
3. SSH in: `ssh root@glass.local` (password: `glass` — change it).
4. The `glass-agent` management API is available immediately on port 8080.

> **Note:** If no `wpa_supplicant.conf` was provided and no ethernet is connected,
> the device has no network. Connect ethernet, then use `POST /network/wifi` to
> configure WiFi.

---

## HTTP Management API

All endpoints are served on `:8080`.

| Method   | Path                  | Description |
|----------|-----------------------|-------------|
| `GET`    | `/status`             | JSON: PID, uptime, restart count |
| `GET`    | `/logs`               | Last 2000 lines of glass output. `?follow=true` streams live. |
| `POST`   | `/ota`                | JSON `{"url":"...","sha256":"<hex>"}`. Replaces `/usr/bin/glass` and restarts. |
| `POST`   | `/config`             | Upload `config.yaml` → restarts glass. |
| `POST`   | `/secrets`            | Upload `secrets.yaml` → restarts glass. |
| `POST`   | `/assets/{name}`      | Upload a file to `/data/assets/`. |
| `DELETE` | `/assets/{name}`      | Delete a file from `/data/assets/`. |
| `POST`   | `/modules/{name}`     | Upload a `.wasm` module to `/data/modules/`. |
| `DELETE` | `/modules/{name}`     | Delete a module from `/data/modules/`. |
| `POST`   | `/os-update`          | JSON `{"url":"..."}`. Downloads and installs a RAUC bundle. Reboot to apply. |
| `GET`    | `/os-status`          | RAUC slot status: active slot, versions, boot state. |
| `POST`   | `/reboot`             | Gracefully triggers a system reboot. |

---

## Common Operations

### Check status
```sh
curl http://glass.local:8080/status
```

### View logs
```sh
# Last 2000 lines
curl http://glass.local:8080/logs

# Stream live
curl http://glass.local:8080/logs?follow=true
```

### OTA update
```sh
curl -X POST http://glass.local:8080/ota \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/glasslabs/looking-glass/releases/download/v1.2.3/glass-v1.2.3-linux-arm64.zip","sha256":"<hex>"}'
```

### Upload config
```sh
curl -X POST http://glass.local:8080/config \
  --data-binary @config.yaml
```

### Upload an asset
```sh
curl -X POST http://glass.local:8080/assets/background.jpg \
  --data-binary @background.jpg
```

### Upload a module
```sh
curl -X POST http://glass.local:8080/modules/clock.wasm \
  --data-binary @clock.wasm
```

### Change WiFi

Drop a `wpa_supplicant.conf` on the FAT32 `/boot` partition (visible from any OS)
and reboot. The file is applied automatically and removed from `/boot` so the PSK is
not left on a readable partition. Dropping a file on `/boot` always wins — use this
to update credentials too.

Alternatively SSH in and write `/data/config/wpa_supplicant.conf` directly, then run
`wpa_cli -i wlan0 reconfigure`.

### OS update (RAUC)

```sh
curl -X POST http://glass.local:8080/os-update \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/glasslabs/os/releases/download/v1.2.3/glassos-v1.2.3-rpi4.raucb"}'

# Then reboot to apply
curl -X POST http://glass.local:8080/reboot
```

### Check OS slot status

```sh
curl http://glass.local:8080/os-status
```

---

## Buildroot Config

### Inspect / change config
```sh
make menuconfig-rpi4      # Buildroot packages
make linux-menuconfig-rpi4  # Kernel options
```

### Save changes
```sh
make savedefconfig-rpi4
```

### Clean build output
```sh
make clean-rpi4    # one board
make clean-all     # everything
```

---

## Adding a New Board

1. Copy an existing board directory: `cp -r buildroot-external/board/rpi4 buildroot-external/board/myboard`
2. Edit `config.txt`, `cmdline.txt`, `linux.config`, and `genimage.cfg` for the new hardware.
3. Copy a defconfig: `cp buildroot-external/configs/glassos_rpi4_defconfig buildroot-external/configs/glassos_myboard_defconfig`
4. Edit the defconfig: update `BR2_LINUX_KERNEL_DEFCONFIG`, config fragment path, and post-build/image script paths.
5. Add `myboard` to the `BOARDS` list in `Makefile`.
6. Add `myboard` to the matrix in `.github/workflows/build.yml`.
