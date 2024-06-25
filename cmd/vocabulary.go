package main

import "net/http"

func (app *application) createVocabulary(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Vocabulary     int32  `json:"vocabulary"`
		TargetLanguage string `json:"target_language"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := app.contextGetUser(r)

	err = app.models.Vocabulary.Insert(user.Id.String(), input.Vocabulary, input.TargetLanguage)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.render.JSON(w, 201, "ok")
}

func (app *application) getVocabulary(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	data, err := app.models.Vocabulary.GetByUser(user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 200, data)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
