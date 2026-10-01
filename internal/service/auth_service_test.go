package service_test

import (
	"testing"

	"github.com/Tnembull/go-core-backend/internal/config"
	"github.com/Tnembull/go-core-backend/internal/model"
	"github.com/Tnembull/go-core-backend/internal/repository"
	"github.com/Tnembull/go-core-backend/internal/service"
)

func TestAuthService_RegisterAndLogin(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret-key-12345",
		JWTExpiration: 1,
	}
	repo := repository.NewInMemoryUserRepository()
	authSvc := service.NewAuthService(repo, cfg)

	// 1. Register User
	registerReq := model.RegisterRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
		Role:     "admin",
	}

	user, err := authSvc.Register(registerReq)
	if err != nil {
		t.Fatalf("expected no error on register, got %v", err)
	}
	if user.Email != registerReq.Email {
		t.Errorf("expected email %s, got %s", registerReq.Email, user.Email)
	}

	// 2. Duplicate Register should fail
	_, err = authSvc.Register(registerReq)
	if err == nil {
		t.Fatal("expected error on duplicate register, got nil")
	}

	// 3. Login with correct credentials
	loginReq := model.LoginRequest{
		Email:    "test@example.com",
		Password: "SecurePassword123!",
	}
	authResp, err := authSvc.Login(loginReq)
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if authResp.Token == "" {
		t.Error("expected non-empty token")
	}

	// 4. Validate Token
	claims, err := authSvc.ValidateToken(authResp.Token)
	if err != nil {
		t.Fatalf("expected token validation to pass, got %v", err)
	}
	if claims.Email != registerReq.Email || claims.Role != "admin" {
		t.Errorf("unexpected claims: %+v", claims)
	}

	// 5. Login with invalid password
	badLogin := model.LoginRequest{
		Email:    "test@example.com",
		Password: "WrongPassword!",
	}
	_, err = authSvc.Login(badLogin)
	if err != service.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}
