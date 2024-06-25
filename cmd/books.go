package main

import "net/http"

func (app *application) createBook(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title string `json:"title"`
		Description string `json:"description"`
		Pages string `json:"pages"`
		ReadPages string `json:"read_pages"`
		ReadType string `json:"read_type"`
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

	app.render.JSON(w, 201, "Ok")
}

func (app *application) getBook(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	data, err := app.models.Book.GetByUser(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.render.JSON(w, 200, data)
}
