# WiFi Provisioning

GlassOS reads NetworkManager connection files from the boot partition at startup and
installs them before NetworkManager starts. No AP, no HTTP request — just a file on
the SD card.

## How it works

The `glassos-net-config` systemd service runs before `NetworkManager.service`. It looks
for a `system-connections/` directory on the boot partition (`/boot`) and, if present,
copies its contents to `/etc/NetworkManager/system-connections/` with permissions set to
`600`. NetworkManager then connects automatically on startup.

## Provisioning steps

1. Flash the GlassOS image to an SD card.
2. Mount the boot partition on your computer (it is FAT32 and readable on any OS).
3. Create a `system-connections/` directory on the boot partition.
4. Place a `.nmconnection` file for your network inside it (see example below).
5. Unmount the boot partition, insert the SD card, and power on the device.

## Connection file format

NetworkManager keyfile format. For a standard WPA2 network:

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

The `uuid` field must be a valid UUID. Generate one with `uuidgen`.

For a hidden network, add `hidden=true` under `[wifi]`.

## Subsequent reboots

Connection profiles written to `/etc/NetworkManager/system-connections/` survive reboots
via the writable overlay partition. NetworkManager reconnects automatically on every boot.
