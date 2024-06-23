package main

import (
	"errors"
	"fmt"
	"language-tracker/internal/data"
	_ "language-tracker/internal/models"
	"language-tracker/internal/tasks"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func (app *application) createUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.log.Err(err)
		http.Error(w, "Invalid Json", http.StatusBadRequest)
		return
	}

	validate := validator.New()
	err = validate.Struct(input)
	if err != nil {
		errors := err.(validator.ValidationErrors)
		http.Error(w, fmt.Sprintf("Validation error: %s", errors), http.StatusBadRequest)
		return
	}

	userId, token, err := app.models.Users.Insert(input.Username, input.Email, input.Password)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			app.badRequestResponse(w, r, err)
			return

		case errors.Is(err, data.ErrDuplicateUsername):
			app.badRequestResponse(w, r, err)
			return

		default:
			app.serverErrorResponse(w, r, err)
			return
		}

	}

	task, err := tasks.NewMailDeliveryTask(userId, "some:template:id")
	if err != nil {
		app.log.Error().Err(err).Msg("TASK NEW MAIL")
	}
	_, err = app.queue.Enqueue(task)
	if err != nil {
		app.log.Error().Err(err).Msg("TASK QUEUE MAIL")
	}

	w.Header().Add("TOKEN", token)
	app.render.JSON(w, http.StatusOK, map[string]string{"message": "User created with success"})
}

func (app *application) activateAccount(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	uuid, err := uuid.Parse(token)

	if err != nil {
		app.render.JSON(w, http.StatusBadRequest, map[string]string{"error": token})
		return
	}

	err = app.models.Users.TokenCheck(uuid)
	if err != nil {
		app.render.JSON(w, 404, map[string]string{"error": data.ErrUserNotFound.Error()})
		return
	}

	app.render.JSON(w, 200, map[string]string{"message": "Success"})
}

func (app *application) showUser(w http.ResponseWriter, r *http.Request) {
}
