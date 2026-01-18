package middleware

import (
	"2026champs/internal/service"
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	HostIDKey   contextKey = "hostId"
	PlayerIDKey contextKey = "playerId"
	RoomCodeKey contextKey = "roomCode"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	authSvc *service.AuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authSvc *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authSvc: authSvc}
}

// RequireHost validates host JWT from Authorization header
func (m *AuthMiddleware) RequireHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		claims, err := m.authSvc.ValidateHostToken(token)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), HostIDKey, claims.HostID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePlayer validates player JWT from Authorization header or query param
func (m *AuthMiddleware) RequirePlayer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			// Try query param for WebSocket
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
			return
		}

		claims, err := m.authSvc.ValidatePlayerToken(token)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, PlayerIDKey, claims.PlayerID)
		ctx = context.WithValue(ctx, RoomCodeKey, claims.RoomCode)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetHostID extracts host ID from context
func GetHostID(ctx context.Context) string {
	if v := ctx.Value(HostIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// GetPlayerID extracts player ID from context
func GetPlayerID(ctx context.Context) string {
	if v := ctx.Value(PlayerIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// GetRoomCode extracts room code from context
func GetRoomCode(ctx context.Context) string {
	if v := ctx.Value(RoomCodeKey); v != nil {
		return v.(string)
	}
	return ""
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
