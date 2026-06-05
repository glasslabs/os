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
  ├── glassos-data-dirs     → ensure /data/{config,assets,modules} exist
  └── glass-agent           → supervisor + HTTP :80
                                ├── NetworkManager (D-Bus) → WiFi provisioning AP + client
                                └── cage
                                      └── glass run ...
```

### Partitions

All partitions are identified by filesystem label so that device paths never need
to be hardcoded in config, cmdline, or RAUC bundles.

```
Label                FS        Size (Pi4/Pi5)   Mount    Purpose
glassos-boot         FAT32     32 MB / 64 MB    /boot    RPi firmware, U-Boot, DTBs
glassos-kernel0      squashfs  24 MB            —        Kernel slot A
glassos-system0      erofs     256 MB           /        rootfs slot A (active)
glassos-kernel1      squashfs  24 MB            —        Kernel slot B (RAUC target)
glassos-system1      erofs     256 MB           —        rootfs slot B (RAUC target)
glassos-bootstate    raw       8 MB             —        U-Boot A/B boot state
glassos-overlay      ext4      96 MB            —        Writable overlay (mutable /etc, /var)
glassos-data         ext4      1280 MB          /data    config, assets, modules (never wiped)
```

See [Documentation/partitions.md](Documentation/partitions.md) for details.

---

## Prerequisites

**Host packages (Ubuntu/Debian):**
```sh
sudo apt-get install -y \
  automake bash bc binutils bison build-essential bzip2 cpio file \
  flex genext2fs genimage gettext git help2man libncurses-dev libssl-dev \
  make patch perl python3 python3-setuptools rsync texinfo unzip wget
```

**Or use the Docker build environment** (no host dependencies needed beyond Docker):
```sh
make docker-build          # build the glassos-builder image once
make docker-run-rpi4       # build Pi 4 image inside Docker
make docker-run-rpi5       # build Pi 5 image inside Docker
```

> The `docker-run-*` targets use a named Docker volume (`glassos-output-rpi4` /
> `glassos-output-rpi5`) for Buildroot's output directory. This keeps
> container-compiled host tools isolated from any native host build, preventing
> `Exec format error` when switching between native and Docker builds.
> The output image is compressed: `sdcard.img.xz`. Decompress before flashing:
> `xz -d sdcard.img.xz`.

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

### Looking Glass version and variant

Two Makefile variables control which `glass` binary is downloaded and embedded:

| Variable | Default   | Description |
|---|-----------|---|
| `GLASS_VERSION` | `v2.0.1`  | looking-glass release tag to download |
| `GLASS_VARIANT` | `wayland` | Gio backend: `wayland` (no X11 dep), `x11`, or `full` |

Both must be kept in sync with their Buildroot counterparts in the defconfig
(`BR2_PACKAGE_GLASS_VERSION`, `BR2_PACKAGE_GLASS_VARIANT`) so that Buildroot
tracks the correct version metadata.

Override on the command line without touching any file:
```sh
make build-rpi4 GLASS_VERSION=v2.1.0 GLASS_VARIANT=wayland
```

Or update the defaults in `Makefile` (and the matching `BR2_PACKAGE_GLASS_*`
values in `buildroot-external/configs/glassos_<board>_defconfig`) before
pushing a release tag so CI picks them up automatically.

> **GlassOS uses the `wayland` variant** — it boots directly into cage (a pure
> Wayland compositor) with no X11 stack present, so `wayland` is the correct
> choice. Use `x11` or `full` only for custom images that include an X server.

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
```

---

## First Boot

1. Insert SD card, connect display and power.
2. If no network connection is available, `glass-agent` automatically starts an open
   WiFi access point named **`GlassOS-Setup`**.
3. Connect to `GlassOS-Setup` from your phone or laptop, then POST WiFi credentials
   to the API (see [WiFi Provisioning](#wifi-provisioning) below).
4. Once the device connects to your network, the AP is automatically removed.
5. Find the device IP via your router's DHCP table (hostname: `glass`).
6. SSH in: `ssh root@<ip>` (password: `glass` — change it).
7. The `glass-agent` management API is available on port 80.

---

## WiFi Provisioning

`glass-agent` handles WiFi provisioning automatically via NetworkManager over D-Bus.

On startup, if NetworkManager reports no active (non-loopback) connections, `glass-agent`
creates and activates an open WiFi access point named **`GlassOS-Setup`** (IPv4 `shared`
mode, no password). You can then connect to that AP and POST credentials to the API:

```sh
curl -X POST http://192.168.4.1/network/wifi \
  -H 'Content-Type: application/json' \
  -d '{"ssid":"MyNetwork","password":"mypassword"}'
```

The agent attempts to connect to the supplied network. If the connection is established
within 30 seconds the AP is deactivated and removed, and any previous WiFi profile for
the same device is cleaned up. If the connection fails or times out, the new profile is
removed and an error is returned so you can try again.

See [Documentation/wifi-provisioning.md](Documentation/wifi-provisioning.md) for full details.

---

## HTTP Management API

All endpoints are served on `:80`.

| Method   | Path                    | Description |
|----------|-------------------------|-------------|
| `GET`    | `/glass/status`         | JSON: PID, uptime, restart count. |
| `GET`    | `/glass/logs`           | Last 2000 lines of glass output. `?follow=true` streams live. |
| `POST`   | `/glass/restart`        | Restarts the Glass process. |
| `POST`   | `/glass/update`         | JSON `{"url":"...","sha256":"<hex>"}`. Replaces `/usr/lib/glass/glass` and restarts. |
| `GET`    | `/glass/config`         | Returns the current `config.yaml`. 404 if not yet written. |
| `POST`   | `/glass/config`         | Upload `config.yaml`. Restart Glass to apply. |
| `POST`   | `/glass/secrets`        | Upload `secrets.yaml`. Restart Glass to apply. |
| `GET`    | `/glass/assets`         | JSON array of asset filenames in `/data/assets/`. |
| `GET`    | `/glass/assets/{name}`  | Download a file from `/data/assets/`. |
| `POST`   | `/glass/assets/{name}`  | Upload a file to `/data/assets/`. |
| `DELETE` | `/glass/assets/{name}`  | Delete a file from `/data/assets/`. |
| `POST`   | `/network/wifi`         | JSON `{"ssid":"...","password":"..."}`. Connects to a WiFi network; removes the provisioning AP on success. |
| `POST`   | `/os/update`            | JSON `{"url":"..."}`. Downloads and installs a RAUC bundle. Reboot to apply. |
| `GET`    | `/os/status`            | RAUC slot status: active slot, versions, boot state. |
| `POST`   | `/os/reboot`            | Gracefully triggers a system reboot. |

---

## Common Operations

### Check status
```sh
curl http://glass.local/glass/status
```

### View logs
```sh
curl http://glass.local/glass/logs
curl http://glass.local/glass/logs?follow=true   # stream live
```

### Restart Glass
```sh
curl -X POST http://glass.local/glass/restart
```

### Provision WiFi
```sh
# While connected to the GlassOS-Setup AP (device IP is 192.168.4.1)
curl -X POST http://192.168.4.1/network/wifi \
  -H 'Content-Type: application/json' \
  -d '{"ssid":"MyNetwork","password":"mypassword"}'
```

### Update Glass binary (OTA)
```sh
curl -X POST http://glass.local/glass/update \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/glasslabs/looking-glass/releases/download/v1.2.3/glass-v1.2.3-linux-arm64-wayland.zip","sha256":"<hex>"}'
```

### Upload config
```sh
curl -X POST http://glass.local/glass/config --data-binary @config.yaml
```

### View current config
```sh
curl http://glass.local/glass/config
```

### Upload an asset
```sh
curl -X POST http://glass.local/glass/assets/background.jpg --data-binary @background.jpg
```

### List assets
```sh
curl http://glass.local/glass/assets
```

### OS update (RAUC)
```sh
curl -X POST http://glass.local/os/update \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/glasslabs/os/releases/download/v1.2.3/glassos-v1.2.3-rpi4.raucb"}'

# Then reboot to apply
curl -X POST http://glass.local/os/reboot
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

The **build** workflow accepts optional `glass_version` and `glass_variant` inputs
(defaulting to the Makefile values) so any combination can be tested without
changing code.

For a **release**, update `GLASS_VERSION` (and optionally `GLASS_VARIANT`) in
`Makefile` and the matching `BR2_PACKAGE_GLASS_*` values in the defconfigs, then
push a tag. CI uses the Makefile defaults to download the correct binary.

The `RAUC_SIGNING_KEY` repository secret must contain the private key matching
`buildroot-external/ota/dev-ca.pem`. CI writes it to a temporary file, passes it
to Buildroot via `GLASSOS_RAUC_KEY`, and removes it after the build. Local builds
fall back to the committed dev key automatically. The certificate expires
**2026-06-23**; run `buildroot-external/ota/gen-dev-key.sh` to regenerate it
before then.
