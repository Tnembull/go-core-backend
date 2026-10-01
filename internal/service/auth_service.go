package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/Tnembull/go-core-backend/internal/config"
	"github.com/Tnembull/go-core-backend/internal/model"
	"github.com/Tnembull/go-core-backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrTokenExpired            = errors.New("token has expired")
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalid2FACode          = errors.New("invalid two-factor authentication code")
	Err2FANotEnabled           = errors.New("two-factor authentication is not enabled")
	ErrInvalidTokenType        = errors.New("invalid token type for this operation")
)

type AuthResponse struct {
	MFARequired  bool   `json:"mfa_required"`
	TempToken    string `json:"temp_token,omitempty"`
	AccessToken  string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"` // in seconds
}

type TwoFactorSetupResponse struct {
	Secret     string   `json:"secret"`
	QRCodeURL  string   `json:"otpauth_url"`
	Recovery   []string `json:"recovery_codes"`
}

type AuthService interface {
	Register(req model.RegisterRequest) (*model.User, error)
	Login(req model.LoginRequest) (*AuthResponse, error)
	Setup2FA(userID string) (*TwoFactorSetupResponse, error)
	Enable2FA(userID, passcode string) error
	Verify2FA(tempToken, passcode string) (*AuthResponse, error)
	RefreshToken(refreshToken string) (*AuthResponse, error)
	Logout(refreshToken string) error
	ValidateToken(tokenStr string, expectedType string) (*model.JWTClaims, error)
	GetUserByID(userID string) (*model.User, error)
}

type authService struct {
	userRepo repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(repo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: repo,
		cfg:      cfg,
	}
}

func (s *authService) Register(req model.RegisterRequest) (*model.User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	role := req.Role
	if role == "" {
		role = model.RoleViewer
	}

	user := &model.User{
		ID:               uuid.New().String(),
		Email:            req.Email,
		Password:         string(hashedPassword),
		Role:             role,
		TwoFactorEnabled: false,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(req model.LoginRequest) (*AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// If 2FA is enabled, issue a temporary pre-auth token
	if user.TwoFactorEnabled {
		tempToken, err := s.generateToken(user, "2fa_preauth", 5*time.Minute)
		if err != nil {
			return nil, err
		}
		return &AuthResponse{
			MFARequired: true,
			TempToken:   tempToken,
		}, nil
	}

	// Normal Login without 2FA
	return s.issueTokenPair(user)
}

func (s *authService) Setup2FA(userID string) (*TwoFactorSetupResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GoCoreBackend",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate totp key: %w", err)
	}

	// Generate 4 recovery codes
	recoveryCodes := make([]string, 4)
	for i := 0; i < 4; i++ {
		bytes := make([]byte, 4)
		_, _ = rand.Read(bytes)
		recoveryCodes[i] = hex.EncodeToString(bytes)
	}

	// Save secret temporarily on user model
	user.TwoFactorSecret = key.Secret()
	user.TwoFactorRecoveryCodes = recoveryCodes
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return &TwoFactorSetupResponse{
		Secret:     key.Secret(),
		QRCodeURL:  key.URL(),
		Recovery:   recoveryCodes,
	}, nil
}

func (s *authService) Enable2FA(userID, passcode string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	if user.TwoFactorSecret == "" {
		return errors.New("2fa setup has not been initiated")
	}

	valid := totp.Validate(passcode, user.TwoFactorSecret)
	if !valid {
		return ErrInvalid2FACode
	}

	user.TwoFactorEnabled = true
	return s.userRepo.Update(user)
}

func (s *authService) Verify2FA(tempToken, passcode string) (*AuthResponse, error) {
	claims, err := s.ValidateToken(tempToken, "2fa_preauth")
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	if !user.TwoFactorEnabled {
		return nil, Err2FANotEnabled
	}

	// Verify TOTP Passcode
	valid := totp.Validate(passcode, user.TwoFactorSecret)
	if !valid {
		// Check recovery codes
		recovered := false
		for i, code := range user.TwoFactorRecoveryCodes {
			if code == passcode {
				user.TwoFactorRecoveryCodes = append(user.TwoFactorRecoveryCodes[:i], user.TwoFactorRecoveryCodes[i+1:]...)
				_ = s.userRepo.Update(user)
				recovered = true
				break
			}
		}
		if !recovered {
			return nil, ErrInvalid2FACode
		}
	}

	return s.issueTokenPair(user)
}

func (s *authService) RefreshToken(refreshTokenStr string) (*AuthResponse, error) {
	claims, err := s.ValidateToken(refreshTokenStr, "refresh")
	if err != nil {
		return nil, err
	}

	// Check if refresh token was revoked
	if s.userRepo.IsTokenRevoked(claims.ID) {
		return nil, repository.ErrInvalidRefreshToken
	}

	// Revoke the old refresh token (Token Rotation)
	_ = s.userRepo.RevokeToken(claims.ID, claims.ExpiresAt.Time)

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	return s.issueTokenPair(user)
}

func (s *authService) Logout(refreshTokenStr string) error {
	claims, err := s.ValidateToken(refreshTokenStr, "refresh")
	if err != nil {
		return nil // idempotent logout
	}
	return s.userRepo.RevokeToken(claims.ID, claims.ExpiresAt.Time)
}

func (s *authService) issueTokenPair(user *model.User) (*AuthResponse, error) {
	accessDuration := time.Duration(s.cfg.JWTExpiration) * time.Hour
	if accessDuration == 0 {
		accessDuration = 24 * time.Hour
	}

	accessToken, err := s.generateToken(user, "access", accessDuration)
	if err != nil {
		return nil, err
	}

	// Refresh token valid for 7 days
	refreshDuration := 7 * 24 * time.Hour
	refreshToken, err := s.generateToken(user, "refresh", refreshDuration)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		MFARequired:  false,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(accessDuration.Seconds()),
	}, nil
}

func (s *authService) generateToken(user *model.User, tokenType string, duration time.Duration) (string, error) {
	claims := &model.JWTClaims{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(), // JTI for revocation tracking
			Subject:   user.ID,
			Issuer:    "go-core-backend",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *authService) ValidateToken(tokenStr string, expectedType string) (*model.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &model.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*model.JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if expectedType != "" && claims.TokenType != expectedType {
		return nil, ErrInvalidTokenType
	}

	return claims, nil
}

func (s *authService) GetUserByID(userID string) (*model.User, error) {
	return s.userRepo.GetByID(userID)
}
