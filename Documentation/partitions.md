# Partition Layout

GlassOS uses a four-partition MBR layout on the SD card. All partitions are
identified by their filesystem label so that references in cmdline, fstab,
and the RAUC config are not sensitive to partition ordering or device paths.

## Partitions

| # | Label             | FS    | Size   | Mount  | Purpose                          |
|---|-------------------|-------|--------|--------|----------------------------------|
| 1 | `glassos-boot`    | FAT32 | 256 MB | `/boot` | U-Boot, firmware, DTBs, cmdline |
| 2 | `glassos-system0` | ext4  | 1.5 GB | `/`    | Active rootfs (slot A)           |
| 3 | `glassos-system1` | ext4  | 1.5 GB | —      | Inactive rootfs (slot B)         |
| 4 | `glassos-data`    | ext4  | 512 MB | `/data` | Persistent data (config, assets, modules) |

## Kernel cmdline

The kernel is told which partition contains the root filesystem via a
filesystem label, making the boot command independent of the device path:

```
root=LABEL=glassos-system0 rootfstype=ext4 rootwait
```

After a successful RAUC update and reboot, U-Boot selects the new active slot
by updating its environment variables; the OS label referred to by the active
slot path changes accordingly.

## RAUC OTA slots

RAUC manages the two system partitions as an A/B pair:

| RAUC slot    | Device path                              | Bootname |
|--------------|------------------------------------------|----------|
| `rootfs.0`   | `/dev/disk/by-label/glassos-system0`     | A        |
| `rootfs.1`   | `/dev/disk/by-label/glassos-system1`     | B        |

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

