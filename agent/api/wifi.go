package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleSetWifi() http.HandlerFunc {
	type wifiRequest struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
	}

	return func(rw http.ResponseWriter, req *http.Request) {
		var wifiReq wifiRequest
		if err := json.NewDecoder(req.Body).Decode(&wifiReq); err != nil {
			http.Error(rw, "invalid request body", http.StatusBadRequest)
			return
		}
		if wifiReq.SSID == "" || wifiReq.Password == "" {
			http.Error(rw, "ssid and password are required", http.StatusBadRequest)
			return
		}

		if err := s.network.SetWiFi(req.Context(), wifiReq.SSID, wifiReq.Password); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusNoContent)
	}
}

