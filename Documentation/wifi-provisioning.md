# WiFi Provisioning

GlassOS uses `glass-agent` to provision WiFi via NetworkManager over D-Bus.
No credentials file on the SD card is required.

## How it works

On startup, `glass-agent` queries NetworkManager for active connections. If no
active (non-loopback) connection exists, it creates and activates an open
802.11 access point:

| Property  | Value           |
|-----------|-----------------|
| SSID      | `GlassOS-Setup` |
| Security  | Open (none)     |
| IPv4      | Shared (NAT)    |

The device is then reachable at `192.168.4.1` on port `80`.

## Provisioning steps

1. Power on the device. If no network is configured the `GlassOS-Setup`
   access point appears within a few seconds.
2. Connect your phone or laptop to `GlassOS-Setup`.
3. POST your WiFi credentials to the agent API:

```sh
curl -X POST http://192.168.4.1:80/network/wifi \
  -H 'Content-Type: application/json' \
  -d '{"ssid":"MyNetworkSSID","password":"mysecretpassword"}'
```

4. The agent adds a WPA2 infrastructure connection and waits up to **30 seconds**
   for it to reach the `Activated` state.
   - **Success** — the AP is deactivated and deleted; any previous WiFi profile
     for the same device is removed. The device is now on your network.
   - **Failure / timeout** — the new connection profile is cleaned up and an
     error is returned. Correct the credentials and try again.

## Subsequent reboots

Once a WiFi connection profile exists, NetworkManager reconnects automatically
on every boot. `glass-agent` detects the active connection at startup and
skips the AP entirely.
