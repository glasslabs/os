package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func (s *Server) handleConfig() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if err := writeRequest(req, filepath.Join(s.dataDir, "config", "config.yaml")); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		s.supervisor.Restart()

		rw.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleSecrets() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		if err := writeRequest(req, filepath.Join(s.dataDir, "config", "secrets.yaml")); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		s.supervisor.Restart()

		rw.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleUploadAsset() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		name := req.PathValue("name")
		dest := filepath.Join(s.dataDir, "assets", filepath.Base(name))

		if err := writeRequest(req, dest); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleDeleteAsset() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		name := req.PathValue("name")
		dest := filepath.Join(s.dataDir, "assets", filepath.Base(name))

		if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusNoContent)
	}
}

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
