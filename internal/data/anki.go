package data

import (
	"context"
	"encoding/json"
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
	IDUser         string    `json:"id_user"`
	Reviewed       string    `json:"reviewed"`
	AddedCards     string    `json:"added_cards"`
	Time           string    `json:"time"`
	TargetLanguage string    `json:"target_language"`
	CreatedAt      time.Time `json:"created_at"`
}

type AnkiData struct {
	Anki               []Anki `json:"rows"`
	DaysAnki           int32  `json:"daysAnki"`
	TotalNewCards      int64  `json:"totalNewCards"`
	TotalReviewed      int64  `json:"totalReviewed"`
	TotalTimeInSeconds string `json:"totalTimeInSeconds"`
}

func (t AnkiModel) Insert(user string, reviewed int32, newCards int32, interval int32, targetLanguage string) error {
	query := "INSERT INTO anki(id_user, reviewed, added_cards, time, target_language) VALUES($1,$2,$3,$4,$5)"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := t.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	args := []any{user, reviewed, newCards, ParseMinutes(interval), targetLanguage}

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
	query := " SELECT time, target_language, created_at, SUM(reviewed) OVER (PARTITION BY reviewed) as totalReviewed, SUM(time) OVER (PARTITION BY time) AS sum_time, SUM(added_cards) OVER (PARTITION BY added_cards) as totalAdded FROM anki WHERE id_user = $1 ORDER BY created_at ASC"

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
		err := rows.Scan(&t, &a.TargetLanguage, &a.CreatedAt, &data.TotalReviewed, &totalTime, &data.TotalNewCards)
		if err != nil {
			return nil, err
		}

		a.Time = FormatTime(t)

		ankis = append(ankis, a)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return nil, err
	}

	var count int32 = 1
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
