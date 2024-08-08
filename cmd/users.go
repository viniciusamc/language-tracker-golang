package main

import (
	"errors"
	"language-tracker/internal/data"
	"language-tracker/internal/tasks"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type DataUser struct {
	User        *data.User           `json:"user"`
	Medias      *data.Medias         `json:"medias"`
	Output      *data.DataOutput     `json:"talk"`
	Anki        *data.AnkiData       `json:"anki"`
	Books       *data.DataBooks      `json:"books"`
	Vocabulary  *data.DataVocabulary `json:"vocabulary"`
	MonthReport *[]data.MonthReport  `json:"month_report"`
	DailyReport *[]data.DailyReport  `json:"daily_report"`
}

func (app *application) createUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	validate := validator.New()
	err = validate.Struct(input)
	if err != nil {
		errors := err.(validator.ValidationErrors)
		app.badRequestResponse(w, r, errors)
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

	task, err := tasks.NewMailDeliveryTask(userId, "email:template", input.Email, token)
	if err != nil {
		app.log.PrintError(err, nil)
	}
	_, err = app.queue.Enqueue(task)
	if err != nil {
		app.log.PrintError(err, nil)
	}

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

func (app *application) showUserSettings(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	app.render.JSON(w, 200, user)
}

func (app *application) userRecoveryPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email" validate:"required,email"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	validate := validator.New()
	err = validate.Struct(input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEmailNotFound):
			app.notFoundResponseSpecified(w, r, err)
			return

		default:
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	task, err := tasks.NewRecoveryPasswordTask(user.Id.String(), "some:template:id", input.Email, "asdas")
	if err != nil {
		app.log.PrintError(err, nil)
	}
	_, err = app.queue.Enqueue(task)
	if err != nil {
		app.log.PrintError(err, nil)
	}

	app.render.JSON(w, 200, user)

}

func (app *application) editUserSettings(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		ReadWordsPerMinute int    `json:"wpm"`
		AverageWordsPage   int    `json:"awp"`
		TargetLanguage     string `json:"TL"`
		DailyGoal          int    `json:"dailyGoal"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	newConfig := data.UserConfig{
		ReadWordsPerMinute:  int32(input.ReadWordsPerMinute),
		AverageWordsPerPage: int32(input.AverageWordsPage),
		TargetLanguage:      input.TargetLanguage,
		DailyGoal:           int32(input.DailyGoal),
	}

	err = app.models.Users.Edit(newConfig, user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.render.JSON(w, 200, "User Config changed with success")
}

func (app *application) showUser(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	monthReport, dailyReport, err := app.models.Users.Report(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	medias, err := app.models.Medias.Get(user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	output, err := app.models.Talks.GetByUser(user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	anki, err := app.models.Anki.GetByUser(user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	books, err := app.models.Book.GetByUser(user)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	vocabulary, err := app.models.Vocabulary.GetByUser(user.Id.String())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	data := DataUser{
		User:        user,
		Medias:      &medias,
		Output:      &output,
		Anki:        anki,
		Books:       books,
		Vocabulary:  vocabulary,
		MonthReport: monthReport,
		DailyReport: dailyReport,
	}

	app.render.JSON(w, 200, data)
}

func (app *application) userExportData(w http.ResponseWriter, r *http.Request) {

}
