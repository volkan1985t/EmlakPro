package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/volkan1985t/EmlakPro/internal/auth"
	"github.com/volkan1985t/EmlakPro/internal/model"
)

type contextKey string

const (
	ContextUserID   contextKey = "user_id"
	ContextUsername contextKey = "username"
	ContextRole     contextKey = "role"
)

type AuthMiddleware struct {
	tokenSvc *auth.TokenService
}

func NewAuthMiddleware(ts *auth.TokenService) *AuthMiddleware {
	return &AuthMiddleware{tokenSvc: ts}
}

// RequireAuth — geçerli JWT olmadan 401 döner
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr == "" {
			jsonError(w, "Oturum açmanız gerekiyor", http.StatusUnauthorized)
			return
		}

		claims, err := m.tokenSvc.ValidateAccessToken(tokenStr)
		if err != nil {
			jsonError(w, "Geçersiz veya süresi dolmuş oturum", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
		ctx = context.WithValue(ctx, ContextUsername, claims.Username)
		ctx = context.WithValue(ctx, ContextRole, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin — admin rolü olmayanları 403 ile reddeder
func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value(ContextRole).(string)
		if role != string(model.RoleAdmin) {
			jsonError(w, "Bu işlem için yetkiniz yok", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// OptionalAuth — token varsa context'e ekler, yoksa devam eder
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr != "" {
			claims, err := m.tokenSvc.ValidateAccessToken(tokenStr)
			if err == nil {
				ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
				ctx = context.WithValue(ctx, ContextUsername, claims.Username)
				ctx = context.WithValue(ctx, ContextRole, claims.Role)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ─── Yardımcılar ─────────────────────────────────────────────────────────────

func extractToken(r *http.Request) string {
	// 1. Authorization: Bearer <token>
	bearer := r.Header.Get("Authorization")
	if strings.HasPrefix(bearer, "Bearer ") {
		return strings.TrimPrefix(bearer, "Bearer ")
	}
	// 2. Cookie: access_token
	cookie, err := r.Cookie("access_token")
	if err == nil {
		return cookie.Value
	}
	return ""
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(`{"success":false,"error":"` + msg + `"}`))
}

// Context yardımcıları
func GetUserID(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(ContextUserID).(int64)
	return v, ok
}

func GetRole(ctx context.Context) string {
	v, _ := ctx.Value(ContextRole).(string)
	return v
}

func IsAdmin(ctx context.Context) bool {
	return GetRole(ctx) == string(model.RoleAdmin)
}
