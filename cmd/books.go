package main

import (
	"errors"
	"language-tracker/internal/data"
	"net/http"
)

func (app *application) createBook(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		Pages          string `json:"pages"`
		ReadPages      string `json:"read_pages"`
		ReadType       string `json:"read_type"`
		TargetLanguage string `json:"target_language"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)

	err = app.models.Book.Insert(user, input.Title, input.Description, input.Pages, input.ReadPages, input.ReadType, input.TargetLanguage)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 201, "Ok")
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getBook(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	data, err := app.models.Book.GetByUser(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 200, data)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateBookProgress(w http.ResponseWriter, r *http.Request) {
	idBook := r.PathValue("idBook")
	var input struct {
		ReadPages int    `json:"read_pages"`
		ReadType  string `json:"read_type"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)

	err = app.models.Book.UpdateBook(user, idBook, input.ReadPages, input.ReadType)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrPageNumberTooLow):
			app.badRequestResponse(w, r, err)
			return

		case errors.Is(err, data.ErrPageNumberTooHigh):
			app.badRequestResponse(w, r, err)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 200, "ok")
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteBook(w http.ResponseWriter, r *http.Request) {
	idBook := r.PathValue("idBook")

	user := app.contextGetUser(r)

	err := app.models.Book.Delete(user, idBook)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 200, "ok")
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
