package main

import (
	"errors"
	"language-tracker/internal/data"
	"language-tracker/internal/tasks"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func (app *application) createMedia(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Url            string `json:"url" validate:"required"`
		Kind           string `json:"type"`
		WatchType      string `json:"watch_type"`
		TargetLanguage string `json:"target_language" validate:"required"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	validate := validator.New()
	err = validate.Struct(input)
	if err != nil {
		app.errorResponse(w, r, 400, err)
		return
	}

	user := app.contextGetUser(r)

	idMedia, videoId, err := app.models.Medias.Insert(user.Id.String(), input.Url, input.Kind, input.WatchType, input.TargetLanguage)
	if err != nil {
		switch {
		case errors.Is(err, data.InvalidUrl):
			app.badRequestResponse(w, r, err)
			return
		default:
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	task, err := tasks.NewTranscriptTask(user.Id.String(), idMedia, videoId, input.TargetLanguage)
	if err != nil {
		app.log.PrintError(err, nil)
	}
	_, err = app.queue.Enqueue(task)
	if err != nil {
		app.log.PrintError(err, nil)
	}

	err = app.render.JSON(w, 201, "Ok")
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getMedia(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	data, err := app.models.Medias.Get(user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 200, data)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) deleteMedia(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	media := r.PathValue("id")

	videoId, targetLanguage, err := app.models.Medias.Delete(user, media)
	if err != nil {
		switch {
		default:
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	task, err := tasks.NewDeleteWordsTask(user.Id.String(), videoId, targetLanguage)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	_, err = app.queue.Enqueue(task)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.render.JSON(w, 200, "Media deleted with success")
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
