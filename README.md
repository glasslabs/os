# GlassOS

Minimal custom Linux OS image for [Looking Glass](https://github.com/glasslabs/looking-glass)
smart mirrors, targeting Raspberry Pi 4 and Pi 5.

Built with [Buildroot](https://buildroot.org). Boots directly into Looking Glass on the
framebuffer via `cage` (a single-application Wayland compositor over DRM/KMS) — no desktop
environment, no X11.

The `supervisor` binary (from [glasslabs/supervisor](https://github.com/glasslabs/supervisor))
supervises the `glass` process and hosts an HTTP API for OTA updates, log access, WiFi
configuration, and config/asset/module uploads.

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
make docker-build   # build the glassos-builder image once
make enter          # drop into an interactive build shell
# then inside the shell:
make build-rpi4     # build Pi 4 image
make build-rpi5     # build Pi 5 image
```

**Clone with submodules:**

```sh
git clone --recurse-submodules https://github.com/glasslabs/os
# or, after cloning:
git submodule update --init
```

### Looking Glass version

The `GLASS_VERSION` Makefile variable controls which `glass` binary is downloaded and
embedded (always the `wayland` variant — no X11 stack is present):

| Variable        | Default  | Description                           |
|-----------------|----------|---------------------------------------|
| `GLASS_VERSION` | `v2.0.5` | looking-glass release tag to download |

It must be kept in sync with `BR2_PACKAGE_GLASS_VERSION` in the defconfig so that
Buildroot tracks the correct version metadata.

Override on the command line without touching any file:

```sh
make build-rpi4 GLASS_VERSION=v2.1.0
```

Or update the default in `Makefile` (and the matching `BR2_PACKAGE_GLASS_VERSION` value in
`buildroot-external/configs/glassos_<board>_defconfig`) before pushing a release tag so CI
picks it up automatically.

### Supervisor version

`GLASSOS_SUPERVISOR_VERSION` controls which `supervisor` binary is downloaded from
[glasslabs/supervisor](https://github.com/glasslabs/supervisor). It must be kept in sync
with `BR2_PACKAGE_GLASSOS_SUPERVISOR_VERSION` in the defconfig.

Override on the command line:

```sh
make build-rpi4 GLASSOS_SUPERVISOR_VERSION=v0.2.0
```

### Caching downloads and compiler output

```sh
# Keep downloads across cleans
make build-rpi4 BR2_DL_DIR=/path/to/shared/dl

# Enable ccache to speed up recompilation
make build-rpi4 BR2_CCACHE_DIR=/path/to/ccache
```

---

## WiFi Provisioning

WiFi is provisioned by placing a NetworkManager connection file on the SD card's boot
partition before first boot. The boot partition is FAT32 and readable from any computer.

Create `system-connections/<name>.nmconnection` on the boot partition:

```ini
[connection]
id=my-network
uuid=c585c544-7776-466d-a60d-85adcb9e2a8c
type=wifi
autoconnect=true

[wifi]
mode=infrastructure
ssid=MyNetwork

[wifi-security]
auth-alg=open
key-mgmt=wpa-psk
psk=mypassword

[ipv4]
method=auto

[ipv6]
addr-gen-mode=stable-privacy
method=auto
```

On boot, `glassos-net-config` copies all files from `boot/system-connections/` to
`/etc/NetworkManager/system-connections/` (with permissions set to `600`) before
NetworkManager starts. NetworkManager then connects automatically.

See [Documentation/wifi-provisioning.md](Documentation/wifi-provisioning.md) for full details.

---

## CI / Releases

| Workflow    | Trigger             | Output                                                              |
|-------------|---------------------|---------------------------------------------------------------------|
| **build**   | `workflow_dispatch` | `sdcard.img.xz` + `.raucb` uploaded as workflow artifacts (14 days) |
| **release** | Tag push            | `sdcard.img.xz` + signed `.raucb` uploaded to the GitHub release    |

The **build** workflow accepts an optional `glass_version` input (defaulting to the Makefile
value) so any version can be tested without changing code. It also accepts a `board` input
to build a single board instead of all boards.

For a **release**, update `GLASS_VERSION` in `Makefile` and the matching
`BR2_PACKAGE_GLASS_VERSION` value in the defconfigs, then push a tag. CI uses the Makefile
defaults to download the correct binary.

The `RAUC_SIGNING_KEY` repository secret must contain the private key matching
`buildroot-external/ota/dev-ca.pem`. CI writes it to a temporary file, passes it to
Buildroot via `GLASSOS_RAUC_KEY`, and removes it after the build. Local builds fall back to
the committed dev key automatically. If the dev certificate has expired, regenerate it with
`buildroot-external/ota/gen-dev-key.sh` (issues a new certificate valid for 10 years).
