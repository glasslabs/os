package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// Status handles GET /status.
func Status(cfg *Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := cfg.Supervisor.Info()

		uptime := ""
		if !info.Started.IsZero() {
			uptime = time.Since(info.Started).Truncate(time.Second).String()
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pid":      info.PID,
			"restarts": info.Restarts,
			"uptime":   uptime,
		})
	}
}

