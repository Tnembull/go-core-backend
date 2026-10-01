package handler

import (
	"net/http"

	"github.com/Tnembull/go-core-backend/internal/middleware"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/Tnembull/go-core-backend/pkg/response"
)

type HealthHandler struct {
	healthSvc service.HealthService
}

func NewHealthHandler(healthSvc service.HealthService) *HealthHandler {
	return &HealthHandler{healthSvc: healthSvc}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	response.JSON(w, http.StatusOK, map[string]string{"status": "UP"}, traceID)
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	response.JSON(w, http.StatusOK, map[string]string{"status": "READY"}, traceID)
}

func (h *HealthHandler) SystemInfo(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	status := h.healthSvc.Check()
	response.JSON(w, http.StatusOK, status, traceID)
}
