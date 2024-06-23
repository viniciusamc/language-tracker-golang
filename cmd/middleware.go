package main

import (
	"errors"
	"language-tracker/internal/data"
	"net/http"
	"strings"
	"time"

	"github.com/pascaldekloe/jwt"
)

func (app *application) authenticate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			app.errorResponse(w, r, 403, "No Token")
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		token := headerParts[1]

		claims, err := jwt.HMACCheck([]byte(token), []byte(app.config.env.JWT_KEY))
		if err != nil {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		if !claims.Valid(time.Now()) {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		if claims.Issuer != "language.tracker" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		if !claims.AcceptAudience("language-tracker") {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}
		userId := claims.Subject

		user, err := app.models.Users.Get(userId)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrUserNotFound):
				app.invalidAuthenticationTokenResponse(w, r)

			default:
				app.serverErrorResponse(w, r, err)
			}

			return
		}

		r = app.contextSetUser(r, user)

		next.ServeHTTP(w, r)
	})
}
