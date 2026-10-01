package handler

import (
	"net/http"

	"github.com/Tnembull/go-core-backend/internal/middleware"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/Tnembull/go-core-backend/pkg/response"
)

type UserHandler struct {
	authSvc service.AuthService
}

func NewUserHandler(authSvc service.AuthService) *UserHandler {
	return &UserHandler{authSvc: authSvc}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized", traceID)
		return
	}

	user, err := h.authSvc.GetUserByID(claims.UserID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user profile not found", traceID)
		return
	}

	response.JSON(w, http.StatusOK, user, traceID)
}

func (h *UserHandler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Welcome to Core Control Plane Admin Resource",
		"scope":   "admin_only",
		"secure":  true,
	}, traceID)
}
