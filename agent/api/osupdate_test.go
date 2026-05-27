package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_HandleOSUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "invalid json body",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing url",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			sup := new(mockSupervisor)
			srv := newServer(t, sup, "", t.TempDir())

			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/os-update", strings.NewReader(test.body))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			assert.Equal(t, test.wantStatus, w.Code)

			sup.AssertExpectations(t)
		})
	}
}

func TestServer_HandleOSStatus(t *testing.T) {
	t.Parallel()

	sup := new(mockSupervisor)
	srv := newServer(t, sup, "", t.TempDir())

	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/os-status", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	// rauc is not installed in the test environment; we expect a 500.
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	sup.AssertExpectations(t)
}

func TestServer_HandleReboot(t *testing.T) {
	t.Parallel()

	sup := new(mockSupervisor)
	srv := newServer(t, sup, "", t.TempDir())

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/reboot", nil)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.ServeHTTP(w, r)
	}()

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond)

	assert.Equal(t, http.StatusNoContent, w.Code)

	sup.AssertExpectations(t)
}

