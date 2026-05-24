package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Config_ handles POST /config — uploads config.yaml to /data/config/ and restarts glass.
func Config_(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := writeRequest(r, filepath.Join(cfg.DataDir, "config", "config.yaml")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cfg.Supervisor.Restart()
		w.WriteHeader(http.StatusNoContent)
	}
}

// Secrets handles POST /secrets — uploads secrets.yaml to /data/config/.
func Secrets(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := writeRequest(r, filepath.Join(cfg.DataDir, "config", "secrets.yaml")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cfg.Supervisor.Restart()
		w.WriteHeader(http.StatusNoContent)
	}
}

// writeRequest reads r.Body and atomically writes it to dest.
func writeRequest(r *http.Request, dest string) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return writeFileAtomic(dest, data, 0o644)
}

// writeFileAtomic writes data to dest atomically using a temp file + rename.
func writeFileAtomic(dest string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	_, err = tmp.Write(data)
	if closeErr := tmp.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	if err = os.Chmod(tmpName, perm); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	if err = os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
