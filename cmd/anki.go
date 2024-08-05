package main

import (
	"net/http"

	"github.com/go-playground/validator/v10"
)

func (app *application) createAnki(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Reviewed       int    `json:"reviewed" validate:"required"`
		NewCards       int    `json:"newCards" validate:"required"`
		Time           int    `json:"time" validate:"required"`
		TargetLanguage string `json:"target_language" validate:"required"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := validator.New()
	err = v.Struct(input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)

	err = app.models.Anki.Insert(user.Id.String(), input.Reviewed, input.NewCards, input.Time, input.TargetLanguage)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 201, &input)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getAnki(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	data, err := app.models.Anki.GetByUser(user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 200, data)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteAnki(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	idAnki := r.PathValue("id")

	err := app.models.Anki.Delete(user, idAnki)
	if err != nil {
		switch {
		default:
			app.serverErrorResponse(w, r, err)
			return
		}
	}
	err = app.render.JSON(w, 200, "Anki Deleted With success")
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
