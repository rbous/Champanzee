package model

import "github.com/golang-jwt/jwt/v5"

// HostClaims are JWT claims for host authentication
type HostClaims struct {
	HostID string `json:"hostId"`
	jwt.RegisteredClaims
}

// PlayerClaims are JWT claims for player room-scoped tokens
type PlayerClaims struct {
	RoomCode string `json:"roomCode"`
	PlayerID string `json:"playerId"`
	jwt.RegisteredClaims
}

// LoginRequest is the request body for host login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is returned after successful login
type LoginResponse struct {
	Token  string `json:"token"`
	HostID string `json:"hostId"`
}
