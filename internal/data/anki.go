package data

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type AnkiModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
}

type Anki struct {
	ID             int64     `json:"id"`
	IDUser         string    `json:"-"`
	Reviewed       int       `json:"reviewed"`
	AddedCards     int       `json:"added_cards"`
	Time           string    `json:"time"`
	TargetLanguage string    `json:"target_language"`
	CreatedAt      time.Time `json:"created_at"`
	Kind           string    `json:"source"`
}

type AnkiData struct {
	Anki               []Anki `json:"anki"`
	DaysAnki           int32  `json:"daysAnki"`
	TotalNewCards      int64  `json:"totalNewCards"`
	TotalReviewed      int64  `json:"totalReviewed"`
	TotalTimeInSeconds string `json:"totalTimeInSeconds"`
}

var (
	ErrAnkiNotFound = errors.New("Anki item not found: The requested Anki item could not be found in the database.")
)

func (t AnkiModel) Insert(user string, reviewed int, newCards int, interval int, targetLanguage string) error {
	query := "INSERT INTO anki(id_user, reviewed, added_cards, time, target_language) VALUES($1,$2,$3,$4,$5)"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := t.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	args := []any{user, reviewed, newCards, ParseMinutes(int32(interval)), targetLanguage}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	t.RDB.Del(context.Background(), "anki:user:"+user)

	return nil
}

func (t AnkiModel) GetByUser(user string) (*AnkiData, error) {
	query := "SELECT id, time::interval, target_language, created_at, SUM(reviewed::int) OVER (PARTITION BY reviewed::int) as totalReviewed, SUM(time::interval) OVER (PARTITION BY time) AS sum_time, SUM(added_cards::integer) OVER (PARTITION BY added_cards) as totalAdded FROM anki WHERE id_user = $1 ORDER BY created_at ASC"

	cache, err := t.RDB.Get(context.Background(), "anki:user:"+user).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if err != redis.Nil {
		var data AnkiData
		err := json.Unmarshal([]byte(cache), &data)
		if err != nil {
			return nil, err
		}
		return &data, nil
	}

	tx, err := t.DB.Begin(context.Background())
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	args := []any{user}

	rows, err := tx.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}

	var ankis []Anki
	var data AnkiData
	var totalTime time.Duration
	for rows.Next() {
		var a Anki
		var t time.Duration
		err := rows.Scan(&a.ID, &t, &a.TargetLanguage, &a.CreatedAt, &a.Reviewed, &totalTime, &a.AddedCards)
		if err != nil {
			return nil, err
		}
		totalTime = +totalTime
		a.Kind = "Anki"

		a.Time = FormatTime(t)

		ankis = append(ankis, a)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return nil, err
	}

	var count int32 = 0
	for i := 1; i < len(ankis); i++ {
		splited := strings.Split(ankis[i].CreatedAt.String(), " ")[0]
		splitb := strings.Split(ankis[i-1].CreatedAt.String(), " ")[0]

		splitedP1, _ := time.Parse("2006-01-02", splited)
		splitbP1, _ := time.Parse("2006-01-02", splitb)

		if splitedP1.Sub(splitbP1) == 24*time.Hour {
			count++
		}
	}

	data.Anki = ankis

	if len(ankis) == 0 {
		data = AnkiData{
			Anki: make([]Anki, 1),
		}
	}

	data.TotalTimeInSeconds = FormatTime(totalTime)
	data.DaysAnki = count
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = t.RDB.Set(context.Background(), "anki:user:"+user, bytes, 0).Err()
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (t AnkiModel) Delete(user *User, id string) error {
	query := "DELETE FROM anki WHERE id_user = $1 AND id = $2"

	tx, err := t.DB.Begin(context.Background())
	if err != nil {
		return nil
	}

	t.RDB.Del(context.Background(), "anki:user:"+user.Id.String())

	args := []any{user.Id.String(), id}
	_, err = tx.Exec(context.Background(), query, args...)

	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return err
	}

	return nil
}
