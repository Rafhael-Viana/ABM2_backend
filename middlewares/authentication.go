package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxRole   ctxKey = "role"
)

type Claims struct {
	UserID string   `json:"user_id"` // subject (id do usuário)
	Role   []string `json:"role"`    // opcional
	jwt.RegisteredClaims
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func bearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization header")
	}
	if parts[1] == "" {
		return "", errors.New("empty token")
	}
	return parts[1], nil
}

// AuthJWT valida HS256 e injeta userId/roles no contexto.
func AuthJWT(secret string) func(http.Handler) http.Handler {
	secretBytes := []byte(secret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := bearerToken(r)
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				// trava o algoritmo
				if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					return nil, errors.New("unexpected signing method")
				}
				return secretBytes, nil
			})
			if err != nil || token == nil || !token.Valid {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}

			// valida exp (RegisteredClaims já faz, mas aqui reforça)
			if claims.ExpiresAt != nil && time.Until(claims.ExpiresAt.Time) <= 0 {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "token expired"})
				return
			}

			if claims.UserID == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token claims"})
				return
			}

			ctx := context.WithValue(r.Context(), CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRoles exige pelo menos 1 role da lista
func RequireRoles(allowed ...string) func(http.Handler) http.Handler {
	set := map[string]struct{}{}
	for _, a := range allowed {
		set[a] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, _ := r.Context().Value(CtxRole).([]string)
			for _, role := range roles {
				if _, ok := set[role]; ok {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		})
	}
}

// Helpers pra ler do contexto
func UserIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(CtxUserID)
	s, ok := v.(string)
	return s, ok
}

func RoleFromContext(ctx context.Context) ([]string, bool) {
	v := ctx.Value(CtxRole)
	s, ok := v.([]string)
	return s, ok
}
