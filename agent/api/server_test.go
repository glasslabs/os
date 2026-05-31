package api_test

import (
	"context"
	"io"
	"testing"

	"github.com/glasslabs/os/agent/api"
	"github.com/glasslabs/os/agent/proc"
	"github.com/hamba/logger/v2"
	"github.com/stretchr/testify/mock"
)

type mockSupervisor struct {
	mock.Mock
}

func (m *mockSupervisor) Restart() {
	m.Called()
}

func (m *mockSupervisor) Info() proc.Info {
	args := m.Called()
	return args.Get(0).(proc.Info)
}

func (m *mockSupervisor) Lines() []string {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.([]string)
	}
	return nil
}

func (m *mockSupervisor) Follow(_ context.Context) <-chan string {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.(<-chan string)
	}
	ch := make(chan string)
	close(ch)
	return ch
}

type mockNetworkManager struct{ mock.Mock }

func (m *mockNetworkManager) SetWiFi(_ context.Context, ssid, password string) error {
	return m.Called(ssid, password).Error(0)
}

func newServer(t *testing.T, sup api.Supervisor, glassBin, dataDir string) *api.Server {
	t.Helper()

	return newServerWithNetwork(t, sup, &mockNetworkManager{}, glassBin, dataDir)
}

func newServerWithNetwork(t *testing.T, sup api.Supervisor, nm api.NetworkManager, glassBin, dataDir string) *api.Server {
	t.Helper()

	log := logger.New(io.Discard, logger.LogfmtFormat(), logger.Info)
	return api.NewServer("", sup, nm, glassBin, dataDir, log)
}
