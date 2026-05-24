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
systemd
  ├── NetworkManager        → WiFi / Ethernet / DHCP
  ├── glassos-wifi-provision → import credentials from /boot on first boot
  ├── glassos-data-dirs     → ensure /data/{config,assets,modules} exist
  └── glass-agent           → supervisor + HTTP :8080
                                └── cage
                                      └── glass run ...
```

### Partitions

All partitions are identified by filesystem label so that device paths never need
to be hardcoded in config, cmdline, or RAUC bundles.

```
Label               FS      Size     Mount    Purpose
glassos-boot        FAT32   256 MB   /boot    U-Boot + kernel + RPi firmware
glassos-system0     ext4    1.5 GB   /        rootfs slot A (active)
glassos-system1     ext4    1.5 GB   —        rootfs slot B (RAUC target)
glassos-data        ext4    512 MB   /data    config, assets, modules (never wiped)
```

See [Documentation/partitions.md](Documentation/partitions.md) for details.

---

## Prerequisites

**Host packages (Ubuntu/Debian):**
```sh
sudo apt-get install -y \
  automake bash bc binutils bison build-essential bzip2 cpio file \
  flex genimage gettext git help2man libncurses-dev libssl-dev \
  make patch perl python3 python3-setuptools rsync texinfo unzip wget
```

**Or use the Docker build environment** (no host dependencies needed beyond Docker):
```sh
make docker-build        # build the glassos-builder image once
docker run --rm -v "$PWD":/build -v "$PWD/buildroot/dl":/cache/dl \
  -w /build glassos-builder make build-rpi4
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

### Looking Glass version

Set the version to embed in the defconfig before building:
```
BR2_PACKAGE_GLASS_VERSION="v1.2.3"
```
Or override at build time without touching the defconfig:
```sh
make build-rpi4 GLASS_VERSION_OVERRIDE=v1.2.3
```

### Caching downloads and compiler output

```sh
# Keep downloads across cleans
make build-rpi4 BR2_DL_DIR=/path/to/shared/dl

# Enable ccache to speed up recompilation
make build-rpi4 BR2_CCACHE_DIR=/path/to/ccache
```

---

## Flashing

```sh
# Flash Pi 4 image to /dev/sdX
make flash BOARD=rpi4 DEV=/dev/sdX

# Flash and write WiFi credentials in one step
make flash BOARD=rpi4 DEV=/dev/sdX SSID="MyNetwork" PSK="mypassword"
```

---

## First Boot

1. Insert SD card, connect display and power.
2. Find the device IP via your router's DHCP table (hostname: `glass`).
3. SSH in: `ssh root@<ip>` (password: `glass` — change it).
4. The `glass-agent` management API is available on port 8080.

---

## WiFi Provisioning

See [Documentation/wifi-provisioning.md](Documentation/wifi-provisioning.md) for full details.

### Before first boot

Mount the FAT32 boot partition (`glassos-boot`) from your computer and place either:

**Option A — NetworkManager keyfile (recommended):**
```ini
# provisioned-wifi.nmconnection
[connection]
id=provisioned-wifi
type=wifi
autoconnect=yes

[wifi]
mode=infrastructure
ssid=MyNetwork

[wifi-security]
key-mgmt=wpa-psk
psk=mypassword

[ipv4]
method=auto

[ipv6]
method=auto
addr-gen-mode=stable-privacy
```

**Option B — legacy `wpa_supplicant.conf`:**
```
network={
    ssid="MyNetwork"
    psk="mypassword"
}
```

Both formats are detected and imported on first boot; the file is then removed from `/boot`.

### After first boot

```sh
nmcli device wifi connect "MyNetwork" password "mypassword"
```

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
curl http://glass.local:8080/logs
curl http://glass.local:8080/logs?follow=true   # stream live
```

### OTA update (glass binary)
```sh
curl -X POST http://glass.local:8080/ota \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/glasslabs/looking-glass/releases/download/v1.2.3/glass-v1.2.3-linux-arm64.zip","sha256":"<hex>"}'
```

### Upload config
```sh
curl -X POST http://glass.local:8080/config --data-binary @config.yaml
```

### Upload an asset or module
```sh
curl -X POST http://glass.local:8080/assets/background.jpg --data-binary @background.jpg
curl -X POST http://glass.local:8080/modules/clock.wasm   --data-binary @clock.wasm
```

### OS update (RAUC)
```sh
curl -X POST http://glass.local:8080/os-update \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/glasslabs/os/releases/download/v1.2.3/glassos-v1.2.3-rpi4.raucb"}'

# Then reboot to apply
curl -X POST http://glass.local:8080/reboot
```

---

## Buildroot Config

```sh
make menuconfig-rpi4        # Buildroot packages
make linux-menuconfig-rpi4  # Kernel options
make savedefconfig-rpi4     # Save changes back to configs/
make clean-rpi4             # Remove build output for one board
make clean-all              # Remove all build output
```

---

## Adding a New Board

See [Documentation/adding-a-board.md](Documentation/adding-a-board.md) for a full walkthrough. The short version:

1. Create `buildroot-external/board/<board-id>/` with `meta`, `config.txt`, `cmdline.txt`, `linux.config`, `genimage.cfg`, and `glassos-hook.sh`.
2. Add thin `post-build.sh` / `post-image.sh` shims that delegate to `scripts/post-build.sh` / `scripts/post-image.sh`.
3. Copy and edit a defconfig in `buildroot-external/configs/`.
4. Add the board ID to `BOARDS` in `Makefile`.
5. Add the board to the matrix in `.github/workflows/build.yml` and `.github/workflows/release.yml`.

---

## CI / Releases

| Workflow | Trigger | Output |
|---|---|---|
| **build** | `workflow_dispatch` | `sdcard.img` uploaded as a workflow artifact (14 days) |
| **release** | Tag push | `sdcard.img` + signed `.raucb` uploaded to the GitHub release |
| **agent** | Push to `main`, PRs | Lint + test the `glass-agent` Go module |

The `RAUC_SIGNING_KEY` repository secret must contain the private key matching
`buildroot-external/ota/dev-ca.pem`. The certificate expires **2026-06-23**; run
`buildroot-external/ota/gen-dev-key.sh` to regenerate it before then.
