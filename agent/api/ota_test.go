package api_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_HandleOTA(t *testing.T) {
	t.Parallel()

	binaryContent := []byte("fake glass binary")
	sum := sha256.Sum256(binaryContent)
	validSHA256 := hex.EncodeToString(sum[:])

	// Serve a fake binary for download.
	downloadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(binaryContent)
	}))
	t.Cleanup(downloadSrv.Close)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBin    bool
	}{
		{
			name:       "downloads and replaces binary",
			body:       `{"url":"` + downloadSrv.URL + `","sha256":"` + validSHA256 + `"}`,
			wantStatus: http.StatusNoContent,
			wantBin:    true,
		},
		{
			name:       "invalid json body",
			body:       "not json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing url",
			body:       `{"sha256":"abc"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing sha256",
			body:       `{"url":"http://example.com/bin"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "sha256 mismatch",
			body:       `{"url":"` + downloadSrv.URL + `","sha256":"000000"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			glassBin := filepath.Join(t.TempDir(), "glass")

			sup := new(mockSupervisor)
			if test.wantBin {
				sup.On("Restart").Return()
			}

			srv := newServer(t, sup, glassBin, t.TempDir())

			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/ota", strings.NewReader(test.body))
			w := httptest.NewRecorder()
			srv.ServeHTTP(w, r)

			assert.Equal(t, test.wantStatus, w.Code)

			if test.wantBin {
				got, err := os.ReadFile(glassBin)
				require.NoError(t, err)
				assert.Equal(t, binaryContent, got)
			}

			sup.AssertExpectations(t)
		})
	}
}

