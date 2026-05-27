package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_HandleConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantRestart bool
	}{
		{
			name:        "writes config and restarts",
			body:        "key: value\n",
			wantStatus:  http.StatusNoContent,
			wantRestart: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			sup := new(mockSupervisor)
			if test.wantRestart {
				sup.On("Restart").Return()
			}

			dataDir := t.TempDir()
			srv := newServer(t, sup, "", dataDir)

			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/config", strings.NewReader(test.body))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			assert.Equal(t, test.wantStatus, w.Code)

			if test.wantStatus == http.StatusNoContent {
				got, err := os.ReadFile(filepath.Join(dataDir, "config", "config.yaml"))
				require.NoError(t, err)
				assert.Equal(t, test.body, string(got))
			}

			sup.AssertExpectations(t)
		})
	}
}

func TestServer_HandleSecrets(t *testing.T) {
	t.Parallel()

	sup := new(mockSupervisor)
	sup.On("Restart").Return()

	dataDir := t.TempDir()
	srv := newServer(t, sup, "", dataDir)

	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/secrets", strings.NewReader("secret: value\n"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)

	got, err := os.ReadFile(filepath.Join(dataDir, "config", "secrets.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "secret: value\n", string(got))

	sup.AssertExpectations(t)
}

func TestServer_HandleUploadAsset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		assetName  string
		body       string
		wantStatus int
	}{
		{
			name:       "uploads asset",
			assetName:  "image.png",
			body:       "fake png data",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "path traversal is rejected by router",
			assetName:  "../escaped.txt",
			body:       "data",
			wantStatus: http.StatusTemporaryRedirect,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			sup := new(mockSupervisor)

			dataDir := t.TempDir()
			srv := newServer(t, sup, "", dataDir)

			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/assets/"+test.assetName, strings.NewReader(test.body))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			assert.Equal(t, test.wantStatus, w.Code)

			if test.wantStatus == http.StatusNoContent {
				got, err := os.ReadFile(filepath.Join(dataDir, "assets", test.assetName))
				require.NoError(t, err)
				assert.Equal(t, test.body, string(got))
			}

			sup.AssertExpectations(t)
		})
	}
}

func TestServer_HandleDeleteAsset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(dataDir string)
		assetName  string
		wantStatus int
	}{
		{
			name: "deletes existing asset",
			setup: func(dataDir string) {
				dir := filepath.Join(dataDir, "assets")
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "image.png"), []byte("data"), 0o644))
			},
			assetName:  "image.png",
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "returns no content when asset does not exist",
			assetName:  "missing.png",
			wantStatus: http.StatusNoContent,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dataDir := t.TempDir()
			if test.setup != nil {
				test.setup(dataDir)
			}

			sup := new(mockSupervisor)
			srv := newServer(t, sup, "", dataDir)

			r := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/assets/"+test.assetName, nil)
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			assert.Equal(t, test.wantStatus, w.Code)

			sup.AssertExpectations(t)
		})
	}
}

