package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glasslabs/os/agent/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ServeHTTP(t *testing.T) {
	t.Parallel()

	s := web.NewServer(false)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://glass.local/", nil)
	require.NoError(t, err)
	rw := httptest.NewRecorder()

	s.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "text/html; charset=utf-8", rw.Header().Get("Content-Type"))
	assert.Contains(t, rw.Body.String(), "glass.local")
	assert.Contains(t, rw.Body.String(), "status-pid")
	assert.NotContains(t, rw.Body.String(), "Setup Mode")
}

func TestServer_ServeHTTP_APMode(t *testing.T) {
	t.Parallel()

	s := web.NewServer(true)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://10.42.0.1/", nil)
	require.NoError(t, err)
	rw := httptest.NewRecorder()

	s.ServeHTTP(rw, req)

	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, "text/html; charset=utf-8", rw.Header().Get("Content-Type"))
	assert.Contains(t, rw.Body.String(), "10.42.0.1")
	assert.Contains(t, rw.Body.String(), "Setup Mode")
	assert.Contains(t, rw.Body.String(), "Connect to Network")
	assert.NotContains(t, rw.Body.String(), `id="ota-url"`)
}


