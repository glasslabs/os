# OTA Updates

GlassOS uses [RAUC](https://rauc.io) for over-the-air updates with an A/B
rootfs scheme.

## How it works

1. The agent (`glass-agent`) exposes a `/ota` HTTP endpoint that accepts a
   signed RAUC bundle (`.raucb` file).
2. RAUC verifies the bundle signature against the keyring at
   `/etc/rauc/keyring.pem`, writes the new rootfs to the inactive slot, and
   marks it for next boot.
3. On the next reboot U-Boot activates the new slot; if it boots successfully
   the slot is confirmed, otherwise U-Boot falls back to the previous slot.

## Signing

### Development builds

The dev CA certificate is committed at `buildroot-external/ota/dev-ca.pem`.
The private key is **not** committed.  To create a bundle locally you need the
corresponding private key:

```sh
rauc bundle \
  --cert=buildroot-external/ota/dev-ca.pem \
  --key=/path/to/dev-ca.key.pem \
  --version=<version> \
  <rootfs.ext4> \
  glassos-<version>-<board>.raucb
```

To regenerate the dev key/cert (e.g., the cert has expired):

```sh
cd buildroot-external/ota
./gen-dev-key.sh
# Commit the new dev-ca.pem.
```

### Release builds

For tagged releases the CI workflow reads the signing key from the
`RAUC_SIGNING_KEY` repository secret and never writes it to disk beyond
a short-lived temp file that is deleted immediately after signing.

The corresponding release certificate must be committed as
`buildroot-external/ota/dev-ca.pem` (or a separate `rel-ca.pem` if you
want to maintain distinct dev and release keyrings).

## RAUC compatible string

Each board has a unique `GLASSOS_COMPATIBLE` value in its `meta` file
(e.g. `glassos-rpi4`). RAUC bundles are board-specific — a bundle built for
`glassos-rpi4` will be rejected on a device running `glassos-rpi5`.

