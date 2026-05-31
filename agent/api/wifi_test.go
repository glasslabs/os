package api_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServer_HandleSetWifi(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		setupMock  func(*mockNetworkManager)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"ssid":"MyNet","password":"s3cr3t"}`,
			setupMock: func(m *mockNetworkManager) {
				m.On("SetWiFi", "MyNet", "s3cr3t").Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid json",
			body:       `{not json`,
			setupMock:  func(_ *mockNetworkManager) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing ssid",
			body:       `{"password":"s3cr3t"}`,
			setupMock:  func(_ *mockNetworkManager) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing password",
			body:       `{"ssid":"MyNet"}`,
			setupMock:  func(_ *mockNetworkManager) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "set wifi error",
			body: `{"ssid":"MyNet","password":"s3cr3t"}`,
			setupMock: func(m *mockNetworkManager) {
				m.On("SetWiFi", "MyNet", "s3cr3t").Return(errors.New("connection failed"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			sup := new(mockSupervisor)
			nm := new(mockNetworkManager)
			test.setupMock(nm)

			srv := newServerWithNetwork(t, sup, nm, "", t.TempDir())

			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/network/wifi", strings.NewReader(test.body))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			require.Equal(t, test.wantStatus, w.Code)
			nm.AssertExpectations(t)
		})
	}
}

