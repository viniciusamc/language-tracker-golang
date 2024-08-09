package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type UserModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
}

type UserConfig struct {
	TargetLanguage      string `json:"TL"`
	ReadWordsPerMinute  int32  `json:"wpm"`
	AverageWordsPerPage int32  `json:"averageWordsPage"`
	DailyGoal           int32  `json:"dailyGoal"`
}

type User struct {
	Id             uuid.UUID `json:"-"`
	Username       string
	Email          string     `json:"-"`
	Password       string     `json:"-"`
	Configs        UserConfig `json:"configs"`
	Email_token    uuid.UUID  `json:"-"`
	Email_verified bool       `json:"-"`
	Created_at     time.Time  `json:"created_at"`
	Updated_at     time.Time
}

type MonthReport struct {
	Month time.Time `json:"month"`
	Hours string    `json:"duration"`
}

type DailyReport struct {
	Day     time.Time `json:"date"`
	Minutes int       `json:"count"`
}

type WordsKnow struct {
	Word     string `json:"word"`
	Amount   string `json:"amount"`
	Language string `json:"language"`
}

var (
	ErrDuplicateEmail    = errors.New("an account with this email already exists")
	ErrDuplicateUsername = errors.New("this username is already taken, please choose another")
	ErrUserNotFound      = errors.New("the specified user could not be found")
	ErrEmailNotFound     = errors.New("the email could not be found")
	WordsNotFound        = errors.New("words not found")
)

func (m UserModel) Insert(username string, email string, password string) (string, string, error) {
	query := "INSERT INTO users(id, username, email, password, configs, email_token) VALUES($1, $2, $3, $4, $5, $6) RETURNING id"
	userConfig := UserConfig{
		TargetLanguage:      "en",
		DailyGoal:           30,
		AverageWordsPerPage: 230,
		ReadWordsPerMinute:  200,
	}
	config, _ := json.Marshal(userConfig)

	tx, err := m.DB.Begin(context.Background())
	if err != nil {
		return "", "", err
	}

	defer tx.Rollback(context.Background())

	passwordHashed, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return "", "", err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return "", "", err
	}

	token, err := uuid.NewRandom()
	if err != nil {
		return "", "", err
	}

	args := []any{id, username, email, passwordHashed, config, token}

	err = tx.QueryRow(context.Background(), query, args...).Scan(&id)
	if err != nil {
		switch {
		case err.Error() == `ERROR: duplicate key value violates unique constraint "users_username_unique" (SQLSTATE 23505)`:
			return "", "", ErrDuplicateUsername

		case err.Error() == `ERROR: duplicate key value violates unique constraint "users_email_unique" (SQLSTATE 23505)`:
			return "", "", ErrDuplicateEmail

		default:
			log.Logger.Error().Err(err).Msg("Error in insert of a new user")
			return "", "", err
		}
	}

	err = tx.Commit(context.Background())
	if err != nil {
		log.Error().Err(err)
	}

	return id.String(), token.String(), nil
}

func (m UserModel) TokenCheck(token uuid.UUID) error {
	query := `UPDATE users SET email_verified = true, email_token = null WHERE email_token = $1 RETURNING username`
	tx, err := m.DB.Begin(context.Background())
	defer tx.Rollback(context.Background())
	if err != nil {
		return ErrDuplicateUsername
	}

	args := []any{token}

	id := ""
	err = tx.QueryRow(context.Background(), query, args...).Scan(&id)
	if err != nil {
		return ErrUserNotFound
	}

	err = tx.Commit(context.Background())
	return nil
}

func (m UserModel) GetByEmail(email string) (*User, error) {
	query := `SELECT id, username, password, configs FROM users WHERE email = $1`

	ctx := context.Background()

	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	args := []any{email}

	var user User

	err = tx.QueryRow(ctx, query, args...).Scan(&user.Id, &user.Username, &user.Password, &user.Configs)
	if err != nil {
		switch {
		case err.Error() == "no rows in result set":
			return nil, ErrEmailNotFound
		}
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (m UserModel) Get(id string) (*User, error) {
	query := `SELECT id, username, password, configs, created_at FROM users WHERE id = $1`
	tx, err := m.DB.Begin(context.Background())
	if err != nil {
		return nil, err
	}

	args := []any{id}

	var user User

	err = tx.QueryRow(context.Background(), query, args...).Scan(&user.Id, &user.Username, &user.Password, &user.Configs, &user.Created_at)
	if err != nil {
		return nil, ErrUserNotFound
	}

	err = tx.Commit(context.Background())

	return &user, nil
}

func (m UserModel) Edit(newConfig UserConfig, id string) error {
	query := "SELECT username, configs FROM users WHERE id = $1"
	tx, err := m.DB.Begin(context.Background())
	if err != nil {
		return err
	}

	var user User

	err = tx.QueryRow(context.Background(), query, id).Scan(&user.Username, &user.Configs)
	if err != nil {
		return err
	}

	if newConfig.DailyGoal != 0 {
		user.Configs.DailyGoal = newConfig.DailyGoal
	}

	if newConfig.AverageWordsPerPage != 0 {
		user.Configs.AverageWordsPerPage = newConfig.AverageWordsPerPage
	}

	if newConfig.TargetLanguage != "" {
		user.Configs.TargetLanguage = newConfig.TargetLanguage
	}

	if newConfig.ReadWordsPerMinute != 0 {
		user.Configs.ReadWordsPerMinute = newConfig.ReadWordsPerMinute
	}

	query = "UPDATE users SET configs = $1 WHERE id = $2"

	args := []any{user.Configs, id}

	_, err = tx.Exec(context.Background(), query, args...)
	if err != nil {
		return err
	}

	tx.Commit(context.Background())

	return nil
}

func (m UserModel) Report(user *User) (*[]MonthReport, *[]DailyReport, error) {
	query := `SELECT
	    DATE_TRUNC('month', created_at) AS month,
	    SUM(time::interval) AS total_time
	FROM (
	SELECT id_user, time::interval, created_at FROM anki
	    UNION ALL
	SELECT id_user, time::interval, created_at FROM medias
	    UNION ALL
	SELECT id_user, time::interval, created_at FROM output
	) AS combined
	WHERE id_user = $1
	GROUP BY month
	ORDER BY month;`

	queryDaily := `
	SELECT 
	DATE_TRUNC('day', created_at) AS day,
	SUM(EXTRACT(EPOCH FROM time::interval) / 60)::integer AS total_minutes
	FROM (
	SELECT id_user, time::interval, created_at FROM anki
		UNION ALL
	SELECT id_user, time::interval, created_at FROM medias
		UNION ALL
	SELECT id_user, time::interval, created_at FROM output
	) as combined
	WHERE id_user = $1
	GROUP BY day
	ORDER BY day;
	`

	ctx := context.Background()

	tx, err := m.DB.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}

	rows, err := tx.Query(ctx, query, user.Id.String())
	if err != nil {
		return nil, nil, err
	}

	var report []MonthReport
	for rows.Next() {
		var t MonthReport
		var d time.Duration
		err := rows.Scan(&t.Month, &d)
		if err != nil {
			return nil, nil, err
		}

		t.Hours = ParseTime(d)
		report = append(report, t)
	}

	rows, err = tx.Query(ctx, queryDaily, user.Id.String())
	if err != nil {
		return nil, nil, err
	}

	var dailyReport []DailyReport
	for rows.Next() {
		var d DailyReport
		err := rows.Scan(&d.Day, &d.Minutes)
		if err != nil {
			return nil, nil, err
		}

		dailyReport = append(dailyReport, d)
	}

	if len(dailyReport) == 0 {
		var d DailyReport
		var m MonthReport
		d.Day = time.Now()
		d.Minutes = 0
		m.Month = time.Now()
		hours, _ := time.ParseDuration("0s")
		m.Hours = ParseTime(hours)
		dailyReport = append(dailyReport, d)
		report = append(report, m)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, nil, err
	}

	return &report, &dailyReport, nil
}

func (m UserModel) GetWords(user *User, language, order string, limit, page, min, max int) (*[]WordsKnow, error) {
	query := fmt.Sprintf(`
	    SELECT w.word,
		   awa.amount,
		   awa.language
	    FROM aux_words_amount awa
	    INNER JOIN words w ON awa.word = w.id
	    WHERE awa.id_user = $1
	      AND awa.language = $2
		AND awa.amount > $5 
		AND awa.amount < $6
	    ORDER BY awa.amount %s
	    LIMIT $3
	    OFFSET $4;
	    `, order)

	offset := (page - 1) * limit

	if offset < 0 {
		offset = 1
	}

	args := []any{user.Id.String(), language, limit, offset, min, max}

	tx, err := m.DB.Begin(context.Background())
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	rows, err := tx.Query(context.Background(), query, args...)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, WordsNotFound
		}
	}

	var wordsKnow []WordsKnow
	for rows.Next() {
		var w WordsKnow
		err := rows.Scan(&w.Word, &w.Amount, &w.Language)
		if err != nil {
			return nil, err
		}

		wordsKnow = append(wordsKnow, w)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return nil, err
	}

	return &wordsKnow, nil
}
