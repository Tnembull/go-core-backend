package handler

import (
	"encoding/json"
	"net/http"

	"github.com/Tnembull/go-core-backend/internal/middleware"
	"github.com/Tnembull/go-core-backend/internal/model"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/Tnembull/go-core-backend/pkg/response"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload", traceID)
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required", traceID)
		return
	}

	user, err := h.authService.Register(req)
	if err != nil {
		response.Error(w, http.StatusConflict, err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusCreated, user, traceID)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload", traceID)
		return
	}

	authResp, err := h.authService.Login(req)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusOK, authResp, traceID)
}

func (h *AuthHandler) Setup2FA(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	claims, ok := r.Context().Value(middleware.UserContextKey).(*model.JWTClaims)
	if !ok || claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized context", traceID)
		return
	}

	setupResp, err := h.authService.Setup2FA(claims.UserID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusOK, setupResp, traceID)
}

func (h *AuthHandler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	claims, ok := r.Context().Value(middleware.UserContextKey).(*model.JWTClaims)
	if !ok || claims == nil {
		response.Error(w, http.StatusUnauthorized, "unauthorized context", traceID)
		return
	}

	var req model.Enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Passcode == "" {
		response.Error(w, http.StatusBadRequest, "valid 6-digit passcode is required", traceID)
		return
	}

	if err := h.authService.Enable2FA(claims.UserID, req.Passcode); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"message": "Two-factor authentication successfully activated",
		"enabled": true,
	}, traceID)
}

func (h *AuthHandler) Verify2FA(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	var req model.Verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TempToken == "" || req.Passcode == "" {
		response.Error(w, http.StatusBadRequest, "temp_token and passcode are required", traceID)
		return
	}

	authResp, err := h.authService.Verify2FA(req.TempToken, req.Passcode)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusOK, authResp, traceID)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	var req model.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "refresh_token is required", traceID)
		return
	}

	authResp, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusOK, authResp, traceID)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())
	var req model.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.RefreshToken != "" {
		_ = h.authService.Logout(req.RefreshToken)
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "logged out successfully and refresh session revoked",
	}, traceID)
}
