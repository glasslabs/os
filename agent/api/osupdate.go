package api

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

func (s *Server) handleOSUpdate() http.HandlerFunc {
	type osUpdateRequest struct {
		URL string `json:"url"`
	}

	return func(rw http.ResponseWriter, req *http.Request) {
		var updateReq osUpdateRequest
		if err := json.NewDecoder(req.Body).Decode(&updateReq); err != nil {
			http.Error(rw, "invalid request body", http.StatusBadRequest)
			return
		}
		if updateReq.URL == "" {
			http.Error(rw, "url is required", http.StatusBadRequest)
			return
		}

		tmp, err := os.CreateTemp("", "glassos-update-*.raucb")
		if err != nil {
			http.Error(rw, "creating temp file", http.StatusInternalServerError)
			return
		}
		tmpName := tmp.Name()
		defer func() { _ = os.Remove(tmpName) }()

		if err = download(req.Context(), tmp, io.Discard, updateReq.URL); err != nil {
			_ = tmp.Close()
			http.Error(rw, fmt.Sprintf("downloading bundle: %v", err), http.StatusBadGateway)
			return
		}
		if err = tmp.Close(); err != nil {
			http.Error(rw, "closing bundle", http.StatusInternalServerError)
			return
		}

		// Use WithoutCancel so a client disconnect does not abort the install.
		//nolint:gosec // Path is controlled.
		out, err := exec.CommandContext(context.WithoutCancel(req.Context()),
			"rauc", "install", tmpName,
		).CombinedOutput()
		if err != nil {
			http.Error(rw, fmt.Sprintf("rauc install failed: %v\n%s", err, out), http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleOSStatus() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		out, err := exec.CommandContext(req.Context(), "rauc", "status", "--output-format=json").Output()
		if err != nil {
			http.Error(rw, fmt.Sprintf("rauc status: %v", err), http.StatusInternalServerError)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write(out)
	}
}

func (s *Server) handleReboot() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusNoContent)
		if f, ok := rw.(http.Flusher); ok {
			f.Flush()
		}

		// Allow the response to reach the client before the system reboots.
		time.Sleep(500 * time.Millisecond)

		_ = exec.CommandContext(context.Background(), "reboot").Run()
	}
}
