package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// HealthResponse represents the health check response body.
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler returns a standard HTTP handler that responds with JSON status "ok".
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := HealthResponse{Status: "ok"}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("failed to encode health check response", "error", err)
		}
	}
}
