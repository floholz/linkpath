package middleware

import (
	"context"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

type ctxKey string

const userKey ctxKey = "user"

// AuthMiddleware validates the pb_auth cookie and stores the user in the request context.
// Unauthenticated requests are redirected to /login.
func AuthMiddleware(app core.App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("pb_auth")
			if err != nil || cookie.Value == "" {
				http.Redirect(w, r, "/~/login", http.StatusFound)
				return
			}
			record, err := app.FindAuthRecordByToken(cookie.Value)
			if err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     "pb_auth",
					Value:    "",
					Path:     "/",
					MaxAge:   -1,
					HttpOnly: true,
				})
				http.Redirect(w, r, "/~/login", http.StatusFound)
				return
			}
			ctx := context.WithValue(r.Context(), userKey, record)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser returns the authenticated user from the request context, or nil.
func GetUser(r *http.Request) *core.Record {
	record, _ := r.Context().Value(userKey).(*core.Record)
	return record
}
