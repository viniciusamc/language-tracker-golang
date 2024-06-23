package main

import (
	"expvar"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (app *application) routes() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.Logger)

	router.Handle("GET /debug/vars", expvar.Handler())
	router.Handle("GET /metrics", promhttp.Handler())

	router.HandleFunc("GET /health", app.healthCheck)

	router.HandleFunc("POST /v1/users", app.createUser)
	router.HandleFunc("GET /v1/users", app.showUser)
	router.HandleFunc("GET /v1/users/token/{token}", app.activateAccount)

	router.HandleFunc("POST /v1/sessions", app.createAuthenticationTokenHandler)

	router.HandleFunc("POST /v1/talk", app.authenticate(app.createTalk))
	router.HandleFunc("GET /v1/talk", app.authenticate(app.getTalk))

	router.HandleFunc("POST /v1/medias", app.authenticate(app.createMedia))

	return router
}


func (app *application) healthCheck(w http.ResponseWriter, r *http.Request) {
	app.render.JSON(w, http.StatusOK, map[string]string{"hello": "json"})
}
