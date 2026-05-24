package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type osUpdateRequest struct {
	URL string `json:"url"`
}

// OSUpdate handles POST /os-update.
// Body: {"url":"https://.../glassos-v1.2.3-rpi4.raucb"}
// Downloads the RAUC bundle and installs it. RAUC handles signature verification,
// slot selection, and bootloader flag setting. Reboot to apply.
func OSUpdate(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req osUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.URL == "" {
			http.Error(w, "url is required", http.StatusBadRequest)
			return
		}

		tmp, err := os.CreateTemp("", "glassos-update-*.raucb")
		if err != nil {
			http.Error(w, "creating temp file", http.StatusInternalServerError)
			return
		}
		tmpName := tmp.Name()
		defer func() { _ = os.Remove(tmpName) }()

		if err = download(r.Context(), tmp, io.Discard, req.URL); err != nil {
			_ = tmp.Close()
			http.Error(w, fmt.Sprintf("downloading bundle: %v", err), http.StatusBadGateway)
			return
		}
		if err = tmp.Close(); err != nil {
			http.Error(w, "closing bundle", http.StatusInternalServerError)
			return
		}

		// Use WithoutCancel so a client disconnect does not abort the install.
		installCtx := context.WithoutCancel(r.Context())
		out, err := exec.CommandContext(installCtx, "rauc", "install", tmpName).CombinedOutput() //nolint:gosec // path is controlled
		if err != nil {
			http.Error(w, fmt.Sprintf("rauc install failed: %v\n%s", err, out), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// OSStatus handles GET /os-status.
// Proxies the output of `rauc status --output-format=json`.
func OSStatus(_ *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := exec.CommandContext(r.Context(), "rauc", "status", "--output-format=json").Output()
		if err != nil {
			http.Error(w, fmt.Sprintf("rauc status: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	}
}

// Reboot handles POST /reboot.
// Flushes the response then triggers a system reboot via the reboot(8) command,
// which allows the init system to run shutdown scripts before restarting.
func Reboot(_ *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Allow the response to reach the client before the system reboots.
		time.Sleep(500 * time.Millisecond)

		_ = exec.CommandContext(context.Background(), "reboot").Run()
	}
}

