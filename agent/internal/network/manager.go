// Package network manages WiFi connections via NetworkManager over D-Bus.
package network

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Wifx/gonetworkmanager/v2"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

const (
	apSSID         = "glassos-setup"
	connectTimeout = 30 * time.Second
)

// Manager manages WiFi provisioning via NetworkManager.
type Manager struct {
	client gonetworkmanager.NetworkManager
	device gonetworkmanager.Device

	apConn       gonetworkmanager.Connection
	apActiveConn gonetworkmanager.ActiveConnection
	connected    chan struct{}

	log *logger.Logger
}

// NewManager returns a new Manager using the first available WiFi device.
func NewManager(log *logger.Logger) (*Manager, error) {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return nil, fmt.Errorf("could not connect to NetworkManager: %w", err)
	}

	dev, err := findWifiDevice(nm)
	if err != nil {
		return nil, fmt.Errorf("could not find WiFi device: %w", err)
	}

	return &Manager{
		client:    nm,
		device:    dev,
		connected: make(chan struct{}, 1),
		log:       log,
	}, nil
}

func findWifiDevice(client gonetworkmanager.NetworkManager) (gonetworkmanager.Device, error) {
	devices, err := client.GetDevices()
	if err != nil {
		return nil, fmt.Errorf("listing devices: %w", err)
	}

	for _, dev := range devices {
		devType, err := dev.GetPropertyDeviceType()
		if err != nil {
			continue
		}

		if devType == gonetworkmanager.NmDeviceTypeWifi {
			return dev, nil
		}
	}

	return nil, errors.New("no WiFi device found")
}

// IsConnected reports whether any non-AP active connection is activated.
func (m *Manager) IsConnected(_ context.Context) (bool, error) {
	state, err := m.client.GetPropertyState()
	if err != nil {
		return false, fmt.Errorf("getting network state: %w", err)
	}

	switch state {
	case gonetworkmanager.NmStateConnectedGlobal,
		gonetworkmanager.NmStateConnectedSite,
		gonetworkmanager.NmStateConnectedLocal:
		return true, nil
	default:
		return false, nil
	}
}

// StartAP creates and activates an open 802.11 access point named "GlassOS-Setup".
func (m *Manager) StartAP(_ context.Context) error {
	existing, err := m.findAPConnection()
	if err != nil {
		return fmt.Errorf("could not check for existing AP profile: %w", err)
	}
	if existing != nil {
		if err = existing.Delete(); err != nil {
			return fmt.Errorf("could not remove stale AP profile: %w", err)
		}
	}

	connMap := map[string]map[string]interface{}{
		"connection": {
			"id":          apSSID,
			"type":        "802-11-wireless",
			"autoconnect": false,
		},
		"802-11-wireless": {
			"mode": "ap",
			"ssid": []byte(apSSID),
			"band": "bg",
		},
		"ipv4": {
			"method": "shared",
			"address-data": []map[string]interface{}{
				{"address": "192.168.4.1", "prefix": uint32(24)},
			},
			"gateway": "192.168.4.1",
		},
		"ipv6": {
			"method": "ignore",
		},
	}

	activeConn, err := m.client.AddAndActivateConnection(connMap, m.device)
	if err != nil {
		return fmt.Errorf("could not start AP: %w", err)
	}

	conn, err := activeConn.GetPropertyConnection()
	if err != nil {
		return fmt.Errorf("could not get AP connection profile: %w", err)
	}

	m.apConn = conn
	m.apActiveConn = activeConn

	m.log.Info("AP started", lctx.Str("ssid", apSSID))

	return nil
}

func (m *Manager) findAPConnection() (gonetworkmanager.Connection, error) {
	settings, err := gonetworkmanager.NewSettings()
	if err != nil {
		return nil, fmt.Errorf("connecting to settings: %w", err)
	}

	all, err := settings.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	for _, conn := range all {
		cfg, err := conn.GetSettings()
		if err != nil {
			continue
		}

		connSection, ok := cfg["connection"]
		if !ok || connSection["id"] != apSSID {
			continue
		}

		wifiSection, ok := cfg["802-11-wireless"]
		if ok && wifiSection["mode"] == "ap" {
			return conn, nil
		}
	}

	return nil, nil
}

// StopAP deactivates and removes the AP connection.
func (m *Manager) StopAP(_ context.Context) error {
	if m.apActiveConn == nil {
		return nil
	}

	if err := m.client.DeactivateConnection(m.apActiveConn); err != nil {
		m.log.Error("Deactivating AP connection", lctx.Err(err))
	}

	if m.apConn != nil {
		if err := m.apConn.Delete(); err != nil {
			return fmt.Errorf("could not remove AP connection profile: %w", err)
		}
	}

	m.apActiveConn = nil
	m.apConn = nil

	m.log.Info("AP stopped")

	return nil
}

// SetWiFi adds and activates a WPA2 infrastructure connection for the given SSID and
// password. On success any prior WiFi connection profiles are removed. On failure the
// new connection is cleaned up and an error is returned.
func (m *Manager) SetWiFi(_ context.Context, ssid, password string) error {
	existing, err := m.listWiFiConnections()
	if err != nil {
		return fmt.Errorf("could not list existing WiFi connections: %w", err)
	}

	connMap := map[string]map[string]interface{}{
		"connection": {
			"id":          ssid,
			"type":        "802-11-wireless",
			"autoconnect": true,
		},
		"802-11-wireless": {
			"mode": "infrastructure",
			"ssid": []byte(ssid),
		},
		"802-11-wireless-security": {
			"key-mgmt": "wpa-psk",
			"psk":      password,
		},
		"ipv4": {
			"method": "auto",
		},
		"ipv6": {
			"method": "auto",
		},
	}

	activeConn, err := m.client.AddAndActivateConnection(connMap, m.device)
	if err != nil {
		return fmt.Errorf("could not add WiFi connection: %w", err)
	}

	newConn, err := activeConn.GetPropertyConnection()
	if err != nil {
		_ = m.client.DeactivateConnection(activeConn)
		return fmt.Errorf("could not get new WiFi connection profile: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	if err = m.waitForConnection(ctx, activeConn); err != nil {
		_ = m.client.DeactivateConnection(activeConn)
		_ = newConn.Delete()
		return fmt.Errorf("could not connect to WiFi: %w", err)
	}

	for _, old := range existing {
		if err = old.Delete(); err != nil {
			m.log.Error("Removing old WiFi connection profile", lctx.Err(err))
		}
	}

	// Successful connection. Notify any watching.
	select {
	case m.connected <- struct{}{}:
	default:
	}

	return nil
}

// WaitForConnection waits for a connection to be made using SetWiFi.
func (m *Manager) WaitForConnection(ctx context.Context, fn func()) {
	select {
	case <-ctx.Done():
	case <-m.connected:
		fn()
	}
}

func (m *Manager) listWiFiConnections() ([]gonetworkmanager.Connection, error) {
	settings, err := gonetworkmanager.NewSettings()
	if err != nil {
		return nil, fmt.Errorf("connecting to settings: %w", err)
	}

	all, err := settings.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	var wifi []gonetworkmanager.Connection
	for _, conn := range all {
		cfg, err := conn.GetSettings()
		if err != nil {
			continue
		}

		connSection, ok := cfg["connection"]
		if !ok {
			continue
		}

		if connSection["type"] != "802-11-wireless" {
			continue
		}

		if m.apConn != nil && conn.GetPath() == m.apConn.GetPath() {
			continue
		}

		wifiSection, ok := cfg["802-11-wireless"]
		if ok && wifiSection["mode"] == "ap" {
			continue
		}

		wifi = append(wifi, conn)
	}

	return wifi, nil
}

// waitForConnection subscribes to active connection state changes until it reaches
// Activated or a terminal failure state. The context deadline acts as the overall timeout.
func (m *Manager) waitForConnection(ctx context.Context, conn gonetworkmanager.ActiveConnection) error {
	receiver := make(chan gonetworkmanager.StateChange, 4)
	exit := make(chan struct{})
	defer close(exit)

	if err := conn.SubscribeState(receiver, exit); err != nil {
		return fmt.Errorf("subscribing to connection state: %w", err)
	}

	// Check the current state first to avoid missing a transition that occurred
	// before the subscription was established.
	state, err := conn.GetPropertyState()
	if err != nil {
		return fmt.Errorf("reading connection state: %w", err)
	}

	switch state {
	case gonetworkmanager.NmActiveConnectionStateActivated:
		return nil
	case gonetworkmanager.NmActiveConnectionStateDeactivated,
		gonetworkmanager.NmActiveConnectionStateDeactivating:
		return errors.New("connection failed or was deactivated")
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for connection: %w", ctx.Err())
		case sc := <-receiver:
			switch sc.State {
			case gonetworkmanager.NmActiveConnectionStateActivated:
				return nil
			case gonetworkmanager.NmActiveConnectionStateDeactivated,
				gonetworkmanager.NmActiveConnectionStateDeactivating:
				return errors.New("connection failed or was deactivated")
			}
		}
	}
}
