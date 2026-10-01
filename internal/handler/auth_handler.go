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
	authSvc service.AuthService
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())

	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", traceID)
		return
	}

	user, err := h.authSvc.Register(req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusCreated, user, traceID)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	traceID := middleware.GetRequestID(r.Context())

	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", traceID)
		return
	}

	authResp, err := h.authSvc.Login(req)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error(), traceID)
		return
	}

	response.JSON(w, http.StatusOK, authResp, traceID)
}
