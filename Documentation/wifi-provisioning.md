# WiFi Provisioning

GlassOS supports two methods for configuring WiFi before or after first boot.

## Before first boot (SD card provisioning)

Copy a credentials file to the FAT32 boot partition (`glassos-boot`, partition 1)
before inserting the SD card for the first time.  On first boot the
`glassos-wifi-provision` service runs, imports the file, removes it from
`/boot`, and hands the connection off to NetworkManager.

### Option A — NetworkManager keyfile (recommended)

Create a file named `<anything>.nmconnection` on the boot partition:

```ini
[connection]
id=my-wifi
type=wifi
autoconnect=yes

[wifi]
mode=infrastructure
ssid=MyNetworkSSID

[wifi-security]
key-mgmt=wpa-psk
psk=mysecretpassword

[ipv4]
method=auto

[ipv6]
method=auto
addr-gen-mode=stable-privacy
```

### Option B — wpa_supplicant.conf (legacy / convenience)

A minimal `wpa_supplicant.conf` on the boot partition is also accepted.  The
provisioning script extracts the SSID and PSK and converts the file to an NM
keyfile automatically:

```
network={
    ssid="MyNetworkSSID"
    psk="mysecretpassword"
}
```

## Using `make flash`

The `make flash` target accepts `SSID=` and `PSK=` and writes an NM keyfile
to the boot partition automatically:

```sh
make flash BOARD=rpi4 DEV=/dev/sdX SSID="MyNetwork" PSK="mypassword"
```

## After first boot

Connect using `nmcli` over SSH (Dropbear runs on port 22):

```sh
nmcli device wifi connect "MyNetworkSSID" password "mysecretpassword"
```

