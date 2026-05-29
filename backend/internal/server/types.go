package server

import "github.com/clagon/port-mapper/backend/internal/service"

// HealthResponse is the JSON payload returned by GET /api/health.
type HealthResponse struct {
	Ok bool `json:"ok"`
}

// ActionResponse is the standard JSON payload returned by mutating endpoints.
type ActionResponse struct {
	Ok bool `json:"ok"`
}

// StatusResponse describes the current discovery and mapping state.
type StatusResponse = service.Status
