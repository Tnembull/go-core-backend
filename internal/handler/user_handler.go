package handler

import (
	"net/http"

	"github.com/Tnembull/go-core-backend/internal/middleware"
	"github.com/Tnembull/go-core-backend/internal/model"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/Tnembull/go-core-backend/pkg/response"
)

type UserHandler struct {
	authService service.AuthService
}

func NewUserHandler(authService service.AuthService) *UserHandler {
	return &UserHandler{authService: authService}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	claims, ok := r.Context().Value(middleware.UserContextKey).(*model.JWTClaims)
	if !ok || claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized context", traceID)
		return
	}

	user, err := h.authService.GetUserByID(claims.UserID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user profile not found", traceID)
		return
	}

	response.JSON(w, http.StatusOK, user, traceID)
}

func (h *UserHandler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	claims := r.Context().Value(middleware.UserContextKey).(*model.JWTClaims)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"scope":       "admin_only",
		"message":     "Welcome to Administrator Mission Control",
		"active_user": claims.Email,
		"role":        claims.Role,
	}, traceID)
}

func (h *UserHandler) WriteUserData(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	claims := r.Context().Value(middleware.UserContextKey).(*model.JWTClaims)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"action":      "users:write",
		"message":     "Authorized to mutate user records",
		"granted_by":  claims.Role,
	}, traceID)
}

func (h *UserHandler) ManageSettings(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	claims := r.Context().Value(middleware.UserContextKey).(*model.JWTClaims)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"action":      "settings:manage",
		"message":     "System-wide configuration settings updated",
		"operator":    claims.Email,
	}, traceID)
}
