package data

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

type UserModel struct {
	DB *pgxpool.Pool
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
	Email          string `json:"-"`
	Password       string `json:"-"`
	Configs        UserConfig `json:"configs"`
	Email_token    uuid.UUID `json:"-"`
	Email_verified bool      `json:"-"`
	Created_at     time.Time
	Updated_at     time.Time
}

var (
	ErrDuplicateEmail    = errors.New("an account with this email already exists")
	ErrDuplicateUsername = errors.New("this username is already taken, please choose another")
	ErrUserNotFound      = errors.New("the specified user could not be found")
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
		return nil, ErrUserNotFound
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (m UserModel) Get(id string) (*User, error) {
	query := `SELECT id, username, password, configs FROM users WHERE id = $1`
	tx, err := m.DB.Begin(context.Background())
	if err != nil {
		return nil, err
	}

	args := []any{id}

	var user User

	err = tx.QueryRow(context.Background(), query, args...).Scan(&user.Id, &user.Username, &user.Password, &user.Configs)
	if err != nil {
		return nil, ErrUserNotFound
	}

	err = tx.Commit(context.Background())

	return &user, nil
}
