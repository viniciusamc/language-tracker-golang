package main

import (
	"errors"
	"language-tracker/internal/data"
	"net/http"
	"time"

	"github.com/pascaldekloe/jwt"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrUserNotFound):
			app.errorResponse(w, r, 400, "Email doesn't exist")
			return
		default:
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		app.errorResponse(w, r, 400, "Wrong Password")
		return
	}

	var claim jwt.Claims
	claim.Subject = user.Id.String()
	claim.Issued = jwt.NewNumericTime(time.Now())
	claim.NotBefore = jwt.NewNumericTime(time.Now())
	claim.Expires = jwt.NewNumericTime(time.Now().Add(24 * 30 * time.Hour))
	claim.Issuer = "language.tracker"
	claim.Audiences = []string{"language-tracker"}

	jwtBytes, err := claim.HMACSign(jwt.HS256, []byte(app.config.env.JWT_KEY))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 201, map[string]string{"user": user.Username, "token": string(jwtBytes)})
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
