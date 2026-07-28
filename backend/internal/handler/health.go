package handler

import (
	"encoding/json"
	"net/http"
)

// HealthResponse is the JSON body for liveness checks.
type HealthResponse struct {
	Status string `json:"status"`
}

// Healthz handles GET /healthz.
func Healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}
