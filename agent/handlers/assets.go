package handlers

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

// UploadAsset handles POST /assets/{name}.
func UploadAsset(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		dest := filepath.Join(cfg.DataDir, "assets", filepath.Base(name))
		if err := writeRequest(r, dest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteAsset handles DELETE /assets/{name}.
func DeleteAsset(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		dest := filepath.Join(cfg.DataDir, "assets", filepath.Base(name))
		if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
