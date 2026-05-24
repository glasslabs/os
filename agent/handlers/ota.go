package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type otaRequest struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// OTA handles POST /ota.
// Body: {"url":"...","sha256":"<hex>"}
// Downloads the binary, verifies its SHA-256, atomically replaces /usr/bin/glass,
// and signals the supervisor to restart.
func OTA(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req otaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.URL == "" || req.SHA256 == "" {
			http.Error(w, "url and sha256 are required", http.StatusBadRequest)
			return
		}

		tmp, err := os.CreateTemp("", "glass-ota-*")
		if err != nil {
			http.Error(w, "creating temp file", http.StatusInternalServerError)
			return
		}
		tmpName := tmp.Name()
		defer func() { _ = os.Remove(tmpName) }()

		h := sha256.New()
		if err = download(r.Context(), tmp, h, req.URL); err != nil {
			_ = tmp.Close()
			http.Error(w, fmt.Sprintf("downloading binary: %v", err), http.StatusBadGateway)
			return
		}
		if err = tmp.Close(); err != nil {
			http.Error(w, "closing temp file", http.StatusInternalServerError)
			return
		}

		got := hex.EncodeToString(h.Sum(nil))
		if got != req.SHA256 {
			http.Error(w, "sha256 mismatch", http.StatusBadRequest)
			return
		}

		if err = os.Chmod(tmpName, 0o755); err != nil {
			http.Error(w, "setting permissions", http.StatusInternalServerError)
			return
		}

		glassBin := "/usr/bin/glass"
		if err = os.Rename(tmpName, glassBin); err != nil {
			http.Error(w, fmt.Sprintf("replacing binary: %v", err), http.StatusInternalServerError)
			return
		}

		cfg.Supervisor.Restart()

		w.WriteHeader(http.StatusNoContent)
	}
}

func download(ctx context.Context, w io.Writer, h io.Writer, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching url: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	_, err = io.Copy(io.MultiWriter(w, h), resp.Body)
	return err
}

