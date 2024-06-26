package main

import (
	"expvar"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (app *application) routes() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.Logger)

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))

	router.Handle("GET /debug/vars", expvar.Handler())
	router.Handle("GET /metrics", promhttp.Handler())

	router.HandleFunc("GET /health", app.healthCheck)

	router.HandleFunc("POST /v1/users", app.createUser)
	router.HandleFunc("GET /v1/user", app.authenticate(app.showUser))
	router.HandleFunc("GET /v1/user/settings", app.authenticate(app.showUserSettings))
	router.HandleFunc("GET /v1/users/token/{token}", app.activateAccount)

	router.HandleFunc("POST /v1/sessions", app.createAuthenticationTokenHandler)

	router.HandleFunc("POST /v1/talk", app.authenticate(app.createTalk))
	router.HandleFunc("GET /v1/talk", app.authenticate(app.getTalk))

	router.HandleFunc("POST /v1/medias", app.authenticate(app.createMedia))
	router.HandleFunc("GET /v1/medias", app.authenticate(app.getMedia))

	router.HandleFunc("POST /v1/anki", app.authenticate(app.createAnki))
	router.HandleFunc("GET /v1/anki", app.authenticate(app.getAnki))

	router.HandleFunc("POST /v1/vocabulary", app.authenticate(app.createVocabulary))
	router.HandleFunc("GET /v1/vocabulary", app.authenticate(app.getVocabulary))

	router.HandleFunc("POST /v1/books", app.authenticate(app.createBook))
	router.HandleFunc("GET /v1/books", app.authenticate(app.getBook))
	router.HandleFunc("PATCH /v1/books/{idBook}", app.authenticate(app.updateBookProgress))
	router.HandleFunc("DELETE /v1/books/{idBook}", app.authenticate(app.deleteBook))
	return router
}

type SystemInfo struct {
	Environment string    `json:"environment"`
	Time        time.Time `json:"time"`
}

func (app *application) healthCheck(w http.ResponseWriter, r *http.Request) {
	app.render.JSON(w, http.StatusOK, map[string]any{"status": "available", "system_info": SystemInfo{
		Environment: app.config.env.Environment,
		Time:        time.Now(),
	}})
}
