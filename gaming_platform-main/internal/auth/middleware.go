package auth

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const userIDKey ctxKey = "user_id"

func Middleware(issuer *TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := ""
			header := r.Header.Get("Authorization")
			if parts := strings.SplitN(header, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenString = parts[1]
			} else if q := r.URL.Query().Get("token"); q != "" {
				// WebSocket handshakes issued from browser JS can't set
				// custom headers, so the match websocket route is reached
				// with the token as a query param instead.
				tokenString = q
			}
			if tokenString == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			claims, err := issuer.Verify(tokenString)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserID extracts the authenticated user id set by Middleware. ok is false
// if called outside a request that passed through Middleware.
func UserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}
