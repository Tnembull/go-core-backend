package service_test

import (
	"testing"
	"time"

	"github.com/Tnembull/go-core-backend/internal/config"
	"github.com/Tnembull/go-core-backend/internal/model"
	"github.com/Tnembull/go-core-backend/internal/repository"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/pquerna/otp/totp"
)

func TestAuthService_FullLifecycleWithRBACAnd2FA(t *testing.T) {
	cfg := &config.Config{
		JWTSecret:     "test-secret-key-1234567890",
		JWTExpiration: 1,
	}
	repo := repository.NewInMemoryUserRepository()
	authSvc := service.NewAuthService(repo, cfg)

	// 1. RBAC Model Permissions Test
	if !model.HasPermission(model.RoleSuperAdmin, "anything") {
		t.Error("superadmin should have wildcard permission")
	}
	if !model.HasPermission(model.RoleAdmin, model.PermissionUsersWrite) {
		t.Error("admin should have users:write permission")
	}
	if model.HasPermission(model.RoleViewer, model.PermissionUsersWrite) {
		t.Error("viewer should not have users:write permission")
	}

	// 2. Register User with role "admin"
	registerReq := model.RegisterRequest{
		Email:    "admin@example.com",
		Password: "SecurePassword123!",
		Role:     model.RoleAdmin,
	}

	user, err := authSvc.Register(registerReq)
	if err != nil {
		t.Fatalf("expected no error on register, got %v", err)
	}
	if user.Role != model.RoleAdmin {
		t.Errorf("expected role %s, got %s", model.RoleAdmin, user.Role)
	}

	// 3. Normal Login (2FA Disabled)
	loginResp, err := authSvc.Login(model.LoginRequest{
		Email:    registerReq.Email,
		Password: registerReq.Password,
	})
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if loginResp.MFARequired {
		t.Error("expected MFARequired to be false initially")
	}
	if loginResp.AccessToken == "" || loginResp.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens")
	}

	// 4. Token Refresh & Rotation Test
	refreshResp, err := authSvc.RefreshToken(loginResp.RefreshToken)
	if err != nil {
		t.Fatalf("expected refresh to succeed, got %v", err)
	}
	if refreshResp.AccessToken == "" || refreshResp.RefreshToken == "" {
		t.Fatal("expected new token pair upon refresh")
	}

	// Reusing old refresh token must fail (Revocation)
	_, err = authSvc.RefreshToken(loginResp.RefreshToken)
	if err == nil {
		t.Fatal("expected old refresh token to be rejected after rotation")
	}

	// 5. 2FA Setup Flow
	setupResp, err := authSvc.Setup2FA(user.ID)
	if err != nil {
		t.Fatalf("expected 2fa setup to succeed, got %v", err)
	}
	if setupResp.Secret == "" || setupResp.QRCodeURL == "" {
		t.Fatal("expected valid 2FA secret and qr code uri")
	}

	// Generate valid TOTP passcode
	passcode, err := totp.GenerateCode(setupResp.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate totp test code: %v", err)
	}

	// Enable 2FA with passcode
	if err := authSvc.Enable2FA(user.ID, passcode); err != nil {
		t.Fatalf("expected 2fa enablement to succeed, got %v", err)
	}

	// 6. Login with 2FA Enabled -> Must require MFA and issue temp token
	mfaLoginResp, err := authSvc.Login(model.LoginRequest{
		Email:    registerReq.Email,
		Password: registerReq.Password,
	})
	if err != nil {
		t.Fatalf("expected login with 2fa to proceed to preauth, got %v", err)
	}
	if !mfaLoginResp.MFARequired || mfaLoginResp.TempToken == "" {
		t.Fatal("expected MFARequired true and non-empty TempToken")
	}
	if mfaLoginResp.AccessToken != "" {
		t.Fatal("access token must NOT be issued before 2FA verification")
	}

	// 7. Verify 2FA with TOTP passcode
	verifyPasscode, _ := totp.GenerateCode(setupResp.Secret, time.Now())
	finalAuthResp, err := authSvc.Verify2FA(mfaLoginResp.TempToken, verifyPasscode)
	if err != nil {
		t.Fatalf("expected 2FA verification to succeed, got %v", err)
	}
	if finalAuthResp.AccessToken == "" {
		t.Fatal("expected final access token after 2FA verification")
	}

	// Validate final Access Token claims
	claims, err := authSvc.ValidateToken(finalAuthResp.AccessToken, "access")
	if err != nil {
		t.Fatalf("expected valid access token, got %v", err)
	}
	if claims.Email != registerReq.Email || claims.Role != model.RoleAdmin {
		t.Errorf("unexpected claims: %+v", claims)
	}
}
