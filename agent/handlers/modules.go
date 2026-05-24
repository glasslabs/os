package handlers

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
)

// UploadModule handles POST /modules/{name}.
func UploadModule(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		dest := filepath.Join(cfg.DataDir, "modules", filepath.Base(name))
		if err := writeRequest(r, dest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteModule handles DELETE /modules/{name}.
func DeleteModule(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		dest := filepath.Join(cfg.DataDir, "modules", filepath.Base(name))
		if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

