package main

import (
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
)

func (app *application) createTalk(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Type string `json:"type" validate:"required"`
		Time string `json:"time" validate:"required"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	timeInt, err := strconv.Atoi(input.Time)
	if err != nil {
		app.errorResponse(w, r, 400, "Invalid Time")
		return
	}

	if timeInt > 4000 || timeInt < 0 {
		app.errorResponse(w, r, 400, "Minutes Max is 4000")
		return
	}

	validate := validator.New()
	err = validate.Struct(input)
	if err != nil {
		errors := err.(validator.ValidationErrors)
		app.badRequestResponse(w, r, errors)
		return
	}

	user := app.contextGetUser(r)


	err = app.models.Talks.Insert(user.Id.String(), input.Type, int16(timeInt), user.Configs.TargetLanguage)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 201, map[string]string{"message": "Success"})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getTalk(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	talks, err := app.models.Talks.GetByUser(user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 200, talks)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
