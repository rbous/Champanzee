package service

import (
	"2026champs/internal/model"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// AuthService handles host and player authentication
type AuthService struct {
	hostUsername string
	hostPassword string
	jwtSecret    []byte
}

// NewAuthService creates a new auth service
func NewAuthService() *AuthService {
	username := os.Getenv("HOST_USERNAME")
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("HOST_PASSWORD")
	if password == "" {
		password = "password123"
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "super-secret-key-change-in-production"
	}

	return &AuthService{
		hostUsername: username,
		hostPassword: password,
		jwtSecret:    []byte(secret),
	}
}

// Login validates credentials and returns a permanent token
func (s *AuthService) Login(username, password string) (*model.LoginResponse, error) {
	if username != s.hostUsername || password != s.hostPassword {
		return nil, ErrInvalidCredentials
	}

	hostID := "host_" + uuid.New().String()[:8]

	claims := &model.HostClaims{
		HostID: hostID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			// No expiry for MVP - permanent token
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		Token:  tokenString,
		HostID: hostID,
	}, nil
}

// ValidateHostToken validates a host JWT and returns claims
func (s *AuthService) ValidateHostToken(tokenString string) (*model.HostClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.HostClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*model.HostClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GeneratePlayerToken creates a room-scoped token for a player
func (s *AuthService) GeneratePlayerToken(roomCode, playerID string) (string, error) {
	claims := &model.PlayerClaims{
		RoomCode: roomCode,
		PlayerID: playerID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24h for room sessions
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidatePlayerToken validates a player JWT and returns claims
func (s *AuthService) ValidatePlayerToken(tokenString string) (*model.PlayerClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.PlayerClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*model.PlayerClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
