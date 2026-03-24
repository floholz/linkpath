package handlers

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"linkpath/internal/render"
)

func LoginPageHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Render(w, "login.html", map[string]any{"Error": ""})
	}
}

func LoginHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.FormValue("email")
		password := r.FormValue("password")

		record, err := app.FindAuthRecordByEmail("_pb_users_auth_", email)
		if err != nil || !record.ValidatePassword(password) {
			tmpl.Render(w, "login.html", map[string]any{"Error": "Invalid email or password."})
			return
		}

		token, err := record.NewAuthToken()
		if err != nil {
			tmpl.Render(w, "login.html", map[string]any{"Error": "Failed to create session. Please try again."})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "pb_auth",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(7 * 24 * time.Hour),
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func RegisterPageHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl.Render(w, "register.html", map[string]any{"Error": ""})
	}
}

func RegisterHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.FormValue("email")
		password := r.FormValue("password")
		passwordConfirm := r.FormValue("password_confirm")

		if password != passwordConfirm {
			tmpl.Render(w, "register.html", map[string]any{"Error": "Passwords do not match."})
			return
		}

		if len(password) < 8 {
			tmpl.Render(w, "register.html", map[string]any{"Error": "Password must be at least 8 characters."})
			return
		}

		usersCollection, err := app.FindCollectionByNameOrId("_pb_users_auth_")
		if err != nil {
			tmpl.Render(w, "register.html", map[string]any{"Error": "Registration failed. Please try again."})
			return
		}

		record := core.NewRecord(usersCollection)
		record.SetEmail(email)
		record.SetPassword(password)

		if err := app.Save(record); err != nil {
			tmpl.Render(w, "register.html", map[string]any{"Error": "Email already in use or invalid."})
			return
		}

		token, err := record.NewAuthToken()
		if err != nil {
			http.Redirect(w, r, "/~/login", http.StatusFound)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "pb_auth",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(7 * 24 * time.Hour),
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func LogoutHandler(app core.App, tmpl *render.Templates) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "pb_auth",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
		})
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
