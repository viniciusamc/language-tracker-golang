package main

import (
	"errors"
	"language-tracker/internal/data"
	"language-tracker/internal/tasks"
	"net/http"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type DataUser struct {
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
		app.log.Err(err)
		http.Error(w, "Invalid Json", http.StatusBadRequest)
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

func (app *application) showUserSettings(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	app.render.JSON(w, 200, user)
}

// func (app *application) showUser(w http.ResponseWriter, r *http.Request) {
// 	user := app.contextGetUser(r)
//
// 	monthReport, dailyReport, err := app.models.Users.Report(user)
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}
//
// 	medias, err := app.models.Medias.Get(user.Id.String())
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}
//
// 	output, err := app.models.Talks.GetByUser(user.Id.String())
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}
//
// 	anki, err := app.models.Anki.GetByUser(user.Id.String())
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}
//
// 	books, err := app.models.Book.GetByUser(user)
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}
//
// 	vocabulary, err := app.models.Vocabulary.GetByUser(user.Id.String())
// 	if err != nil {
// 		app.badRequestResponse(w, r, err)
// 		return
// 	}
//
// 	data := DataUser{
// 		Medias:      &medias,
// 		Output:      &output,
// 		Anki:        anki,
// 		Books:       books,
// 		Vocabulary:  vocabulary,
// 		MonthReport: monthReport,
// 		DailyReport: dailyReport,
// 	}
//
// 	app.render.JSON(w, 200, data)
// }

func (app *application) showUser(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	var wg sync.WaitGroup

	mediasCh := make(chan *data.Medias, 1)
	outputCh := make(chan *data.DataOutput, 1)
	ankiCh := make(chan *data.AnkiData, 1)
	booksCh := make(chan *data.DataBooks, 1)
	vocabularyCh := make(chan *data.DataVocabulary, 1)
	monthReportCh := make(chan *[]data.MonthReport, 1)
	dailyReportCh := make(chan *[]data.DailyReport, 1)
	errCh := make(chan error, 6) 

	wg.Add(1)
	go func() {
		defer wg.Done()
		monthReport, dailyReport, err := app.models.Users.Report(user)
		if err != nil {
			errCh <- err
			return
		}
		monthReportCh <- monthReport
		dailyReportCh <- dailyReport
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		medias, err := app.models.Medias.Get(user.Id.String())
		if err != nil {
			errCh <- err
			return
		}
		mediasCh <- &medias
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		output, err := app.models.Talks.GetByUser(user.Id.String())
		if err != nil {
			errCh <- err
			return
		}
		outputCh <- &output
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		anki, err := app.models.Anki.GetByUser(user.Id.String())
		if err != nil {
			errCh <- err
			return
		}
		ankiCh <- anki
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		books, err := app.models.Book.GetByUser(user)
		if err != nil {
			errCh <- err
			return
		}
		booksCh <- books
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		vocabulary, err := app.models.Vocabulary.GetByUser(user.Id.String())
		if err != nil {
			errCh <- err
			return
		}
		vocabularyCh <- vocabulary
	}()

	go func() {
		wg.Wait()
		close(monthReportCh)
		close(dailyReportCh)
		close(mediasCh)
		close(outputCh)
		close(ankiCh)
		close(booksCh)
		close(vocabularyCh)
		close(errCh)
	}()

	var (
		monthReport  *[]data.MonthReport
		dailyReport  *[]data.DailyReport
		medias       *data.Medias
		output       *data.DataOutput
		anki         *data.AnkiData
		books        *data.DataBooks
		vocabulary   *data.DataVocabulary
		err          error
	)
	for {
		select {
		case monthReport = <-monthReportCh:
		case dailyReport = <-dailyReportCh:
		case medias = <-mediasCh:
		case output = <-outputCh:
		case anki = <-ankiCh:
		case books = <-booksCh:
		case vocabulary = <-vocabularyCh:
		case err = <-errCh:
			if err != nil {
				app.badRequestResponse(w, r, err)
				return
			}
		}

		if len(monthReportCh) == 0 && len(dailyReportCh) == 0 && len(mediasCh) == 0 && 
		   len(outputCh) == 0 && len(ankiCh) == 0 && len(booksCh) == 0 && len(vocabularyCh) == 0 {
			break
		}
	}

	data := DataUser{
		MonthReport:  monthReport,
		DailyReport:  dailyReport,
		Medias:       medias,
		Output:       output,
		Anki:         anki,
		Books:        books,
		Vocabulary:   vocabulary,
	}

	app.render.JSON(w, 200, data)
}
