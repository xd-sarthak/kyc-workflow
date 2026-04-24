package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"kyc/backend/store"
)

// AuthService handles user authentication and JWT token management.
type AuthService struct {
	userStore *store.UserStore
	jwtSecret []byte
}

// NewAuthService creates a new AuthService.
func NewAuthService(userStore *store.UserStore, jwtSecret string) *AuthService {
	return &AuthService{
		userStore: userStore,
		jwtSecret: []byte(jwtSecret),
	}
}

// Signup creates a new user and returns a JWT token.
func (s *AuthService) Signup(ctx context.Context, email, password, role string) (string, error) {
	if email == "" || password == "" {
		return "", fmt.Errorf("email and password are required")
	}
	if role != "merchant" && role != "reviewer" {
		return "", fmt.Errorf("role must be 'merchant' or 'reviewer'")
	}

	exists, err := s.userStore.UserExists(ctx, email)
	if err != nil {
		slog.Error("signup: failed to check user existence",
			"email", email,
			"error", err,
		)
		return "", fmt.Errorf("failed to check user: %w", err)
	}
	if exists {
		slog.Warn("signup: duplicate email attempt",
			"email", email,
		)
		return "", fmt.Errorf("user with email %q already exists", email)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		slog.Error("signup: bcrypt hash failed", "error", err)
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.userStore.CreateUser(ctx, email, string(hashedPassword), role)
	if err != nil {
		slog.Error("signup: failed to create user",
			"email", email,
			"role", role,
			"error", err,
		)
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	slog.Info("signup: user created",
		"user_id", user.ID,
		"email", email,
		"role", role,
	)

	return s.generateToken(user.ID, user.Role)
}

// Login verifies credentials and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", fmt.Errorf("email and password are required")
	}

	user, err := s.userStore.GetUserByEmail(ctx, email)
	if err != nil {
		slog.Warn("login: user not found",
			"email", email,
		)
		return "", fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		slog.Warn("login: invalid password",
			"email", email,
			"user_id", user.ID,
		)
		return "", fmt.Errorf("invalid credentials")
	}

	slog.Info("login: successful",
		"user_id", user.ID,
		"email", email,
		"role", user.Role,
	)

	return s.generateToken(user.ID, user.Role)
}

// generateToken creates a JWT token with the given claims.
func (s *AuthService) generateToken(userID uuid.UUID, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		slog.Error("jwt: failed to sign token",
			"user_id", userID,
			"error", err,
		)
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken parses and validates a JWT token, returning the user ID and role.
func (s *AuthService) ValidateToken(tokenString string) (uuid.UUID, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return uuid.Nil, "", fmt.Errorf("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("invalid token: missing sub claim")
	}
	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid token: bad user ID")
	}

	role, ok := claims["role"].(string)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("invalid token: missing role claim")
	}

	return userID, role, nil
}
