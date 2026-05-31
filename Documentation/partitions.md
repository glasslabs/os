# Partition Layout

GlassOS uses an 8-partition layout on the SD card. All partitions are
identified by their GPT partition label so that references in cmdline, fstab,
and the RAUC config are not sensitive to device paths.

Pi 4 uses a hybrid MBR/GPT table; Pi 5 uses pure GPT.

## Partitions

| # | Label                | FS       | Size (Pi4 / Pi5) | Mount   | Purpose                              |
|---|----------------------|----------|------------------|---------|--------------------------------------|
| 1 | `glassos-boot`       | FAT32    | 32 MB / 64 MB    | `/boot` | RPi firmware, U-Boot, DTBs, cmdline  |
| 2 | `glassos-kernel0`    | squashfs | 24 MB            | —       | Kernel image, slot A                 |
| 3 | `glassos-system0`    | erofs    | 256 MB           | `/`     | rootfs slot A (active)               |
| 4 | `glassos-kernel1`    | squashfs | 24 MB            | —       | Kernel image, slot B (RAUC target)   |
| 5 | `glassos-system1`    | erofs    | 256 MB           | —       | rootfs slot B (RAUC target)          |
| 6 | `glassos-bootstate`  | raw      | 8 MB             | —       | U-Boot A/B boot state variables      |
| 7 | `glassos-overlay`    | ext4     | 96 MB            | —       | Writable overlay (mutable /etc, /var)|
| 8 | `glassos-data`       | ext4     | 1280 MB          | `/data` | Persistent data (config, assets, modules) |

## Kernel cmdline

The kernel is told which partition contains the root filesystem via a
partition label, making the boot command independent of the device path:

```
root=PARTLABEL=glassos-system0 rootfstype=erofs rootwait
```

After a successful RAUC update and reboot, U-Boot selects the new active slot
by updating its bootstate partition; the root label changes to `glassos-system1`
accordingly.

## RAUC OTA slots

RAUC manages kernel and rootfs as paired A/B slots. The boot partition is also
a RAUC slot so the firmware can be updated independently.

| RAUC slot    | Device path                                   | Bootname |
|--------------|-----------------------------------------------|----------|
| `boot.0`     | `/dev/disk/by-partlabel/glassos-boot`         | —        |
| `kernel.0`   | `/dev/disk/by-partlabel/glassos-kernel0`      | A        |
| `rootfs.0`   | `/dev/disk/by-partlabel/glassos-system0`      | A        |
| `kernel.1`   | `/dev/disk/by-partlabel/glassos-kernel1`      | B        |
| `rootfs.1`   | `/dev/disk/by-partlabel/glassos-system1`      | B        |

The RAUC `system.conf` is generated at build time from
`buildroot-external/ota/system.conf.tmpl` using the board's `GLASSOS_COMPATIBLE`
value from its `meta` file.

## Dev CA certificate

The development CA certificate used to sign RAUC bundles lives at
`buildroot-external/ota/dev-ca.pem`.  The corresponding **private key must
never be committed**; it is ignored by `.gitignore`.

To regenerate a long-lived dev key/cert pair:

```sh
cd buildroot-external/ota
./gen-dev-key.sh
# Commit dev-ca.pem — keep dev-ca.key.pem secret.
```

For production releases the signing key is injected as a CI secret
(`RAUC_SIGNING_KEY`) and is never stored in the repository.
