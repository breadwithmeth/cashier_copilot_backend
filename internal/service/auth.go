package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cashier_copilot_backend/internal/model"
	"cashier_copilot_backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("expired token")
	ErrInactiveUser       = errors.New("inactive user")
	ErrForbidden          = errors.New("forbidden")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidRole        = errors.New("invalid role")
)

// TokenClaims is the signed access token payload.
type TokenClaims struct {
	UserID    int64   `json:"sub"`
	Username  string  `json:"username"`
	Role      string  `json:"role"`
	PosID     *string `json:"pos_id,omitempty"`
	IssuedAt  int64   `json:"iat"`
	ExpiresAt int64   `json:"exp"`
}

// AuthService handles local password auth and signed access tokens.
type AuthService struct {
	userRepo *repository.UserRepo
	secret   []byte
	ttl      time.Duration
	logger   *slog.Logger
}

// NewAuthService creates an AuthService.
func NewAuthService(userRepo *repository.UserRepo, jwtSecret string, ttl time.Duration, logger *slog.Logger) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		secret:   []byte(jwtSecret),
		ttl:      ttl,
		logger:   logger,
	}
}

// EnsureBootstrapAdmin creates an admin user if it does not already exist.
func (s *AuthService) EnsureBootstrapAdmin(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}

	created, err := s.userRepo.InsertIfMissing(ctx, &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         model.RoleAdmin,
		IsActive:     true,
	})
	if err != nil {
		return err
	}

	if created {
		s.logger.Info("bootstrap admin user created", "username", username)
	} else {
		s.logger.Info("bootstrap admin user already exists", "username", username)
	}

	return nil
}

// Authenticate validates username and password, then returns a token.
func (s *AuthService) Authenticate(ctx context.Context, username, password string) (*model.AuthUser, string, int64, error) {
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, "", 0, err
	}
	if user == nil || !user.IsActive {
		return nil, "", 0, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", 0, ErrInvalidCredentials
	}

	authUser := publicUser(user)
	token, expiresAt, err := s.IssueToken(authUser)
	if err != nil {
		return nil, "", 0, err
	}

	return authUser, token, expiresAt, nil
}

// ListUsers returns users for admin management.
func (s *AuthService) ListUsers(ctx context.Context) ([]model.User, error) {
	return s.userRepo.List(ctx)
}

// CreateUser creates a local user with a bcrypt password hash.
func (s *AuthService) CreateUser(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
	if !validRole(req.Role) {
		return nil, ErrInvalidRole
	}

	existing, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash user password: %w", err)
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	user := &model.User{
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         req.Role,
		PosID:        req.PosID,
		IsActive:     isActive,
	}

	id, err := s.userRepo.Insert(ctx, user)
	if err != nil {
		return nil, err
	}
	user.ID = id
	return user, nil
}

// IssueToken creates a signed access token.
func (s *AuthService) IssueToken(user *model.AuthUser) (string, int64, error) {
	now := time.Now()
	expiresAt := now.Add(s.ttl).Unix()

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", 0, err
	}

	claimsJSON, err := json.Marshal(TokenClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		PosID:     user.PosID,
		IssuedAt:  now.Unix(),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", 0, err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := header + "." + payload
	signature := s.sign(unsigned)

	return unsigned + "." + signature, expiresAt, nil
}

// ValidateAccessToken validates token signature, expiry, and current user state.
func (s *AuthService) ValidateAccessToken(ctx context.Context, token string) (*model.AuthUser, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, ErrInactiveUser
	}

	return publicUser(user), nil
}

// UserCanAccessPos checks whether a user can access a cashier terminal POS stream.
func UserCanAccessPos(user *model.AuthUser, posID string) bool {
	if user == nil {
		return false
	}
	if user.Role == model.RoleAdmin || user.Role == model.RoleOperator {
		return true
	}
	if user.Role == model.RoleCashier && user.PosID != nil && *user.PosID == posID {
		return true
	}
	return false
}

func (s *AuthService) parseToken(token string) (*TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	expected := s.sign(unsigned)
	if subtle.ConstantTimeCompare([]byte(parts[2]), []byte(expected)) != 1 {
		return nil, ErrInvalidToken
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims TokenClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.ExpiresAt <= time.Now().Unix() {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

func (s *AuthService) sign(unsigned string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func publicUser(user *model.User) *model.AuthUser {
	return &model.AuthUser{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
		PosID:    user.PosID,
	}
}

func validRole(role string) bool {
	return role == model.RoleAdmin || role == model.RoleOperator || role == model.RoleCashier
}
