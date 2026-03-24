package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"linkpath/internal/handlers"
	"linkpath/internal/middleware"
	"linkpath/internal/render"

	_ "linkpath/migrations"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	app := pocketbase.New()

	var appHTTPAddr string
	app.RootCmd.PersistentFlags().StringVar(&appHTTPAddr, "app-http", envOr("APP_HTTP", "0.0.0.0:8080"), "app HTTP server address")

	// Allow PB_HTTP env var to set PocketBase's own --http flag default.
	if pbHTTP := os.Getenv("PB_HTTP"); pbHTTP != "" {
		for _, sub := range app.RootCmd.Commands() {
			if sub.Name() == "serve" {
				if f := sub.Flags().Lookup("http"); f != nil {
					f.DefValue = pbHTTP
					_ = f.Value.Set(pbHTTP)
				}
				break
			}
		}
	}

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		tmpl, err := render.New(TemplatesFS)
		if err != nil {
			return err
		}

		authMw := middleware.AuthMiddleware(app)
		wrap := func(h http.HandlerFunc) http.Handler { return authMw(h) }

		mux := http.NewServeMux()

		// Static files
		mux.Handle("/static/", http.FileServerFS(StaticFS))

		// Reject common browser-generated paths that must never become nodes.
		notFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
		mux.Handle("/favicon.ico", notFound)
		mux.Handle("/robots.txt", notFound)
		mux.Handle("/.well-known/", notFound)

		// Public routes (prefixed with /~/ to avoid conflicting with linkpath paths)
		mux.HandleFunc("GET /~/login", handlers.LoginPageHandler(app, tmpl))
		mux.HandleFunc("POST /~/login", handlers.LoginHandler(app, tmpl))
		mux.HandleFunc("GET /~/register", handlers.RegisterPageHandler(app, tmpl))
		mux.HandleFunc("POST /~/register", handlers.RegisterHandler(app, tmpl))

		// Auth-required routes
		mux.Handle("POST /~/logout", wrap(handlers.LogoutHandler(app, tmpl)))
		mux.Handle("GET /~/items/add-form", wrap(handlers.AddFormHandler(app, tmpl)))
		mux.Handle("POST /~/items", wrap(handlers.CreateItemHandler(app, tmpl)))
		mux.Handle("GET /~/items/{id}", wrap(handlers.GetItemHandler(app, tmpl)))
		mux.Handle("GET /~/items/{id}/edit", wrap(handlers.EditItemHandler(app, tmpl)))
		mux.Handle("PUT /~/items/{id}", wrap(handlers.UpdateItemHandler(app, tmpl)))
		mux.Handle("DELETE /~/items/{id}", wrap(handlers.DeleteItemHandler(app, tmpl)))
		mux.Handle("POST /~/items/{id}/move", wrap(handlers.MoveItemHandler(app, tmpl)))

		// Catch-all: handles / (landing/home) and all path views
		mux.HandleFunc("/", handlers.PathHandler(app, tmpl, envOr("APP_HOST", "linkpa.th")))

		appServer := &http.Server{
			Addr:    appHTTPAddr,
			Handler: mux,
		}

		app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = appServer.Shutdown(ctx)
			return te.Next()
		})

		go func() {
			log.Printf("App server listening on http://%s", appHTTPAddr)
			if err := appServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("App server error: %v", err)
			}
		}()

		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
