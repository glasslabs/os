package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

// Logs handles GET /logs.
// Query params: ?follow=true streams new lines via chunked response.
func Logs(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		follow := r.URL.Query().Get("follow") == "true"

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Write buffered lines first.
		lines := cfg.Supervisor.Lines()
		if len(lines) > 0 {
			_, _ = fmt.Fprint(w, strings.Join(lines, "\n")+"\n")
		}

		if !follow {
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()

		ch := cfg.Supervisor.Follow(r.Context())
		for line := range ch {
			_, _ = fmt.Fprint(w, line+"\n")
			flusher.Flush()
		}
	}
}
