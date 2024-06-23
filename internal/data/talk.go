package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type TalkModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
}

type Talk struct {
	Id             uuid.UUID
	IdUser         uuid.UUID
	Kind           string
	Time           string
	Summarize      string
	TargetLanguage string
	CreatedAt      time.Time
}

func (t TalkModel) Insert(id string, kind string, minutes int16, targetLanguage string) error {
	query := `INSERT INTO output(id_user, type, time, target_language) VALUES($1,$2,$3,$4)`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := t.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	var min time.Time
	min = min.Add(time.Duration(minutes) * time.Minute)

	args := []any{id, kind, min.Format("15:04:05"), targetLanguage}

	err = t.RDB.Del(ctx, `talk:user:`+id).Err()

	if err != nil {
		println("redis error set")
		return err
	}
	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (t TalkModel) GetByUser(id string) (*[]Talk, error) {
	query := `SELECT type, time, target_language, created_at FROM output WHERE id_user = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cache, err := t.RDB.Get(ctx, `talk:user:` + id ).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if err != redis.Nil {
		var talks []Talk
		err := json.Unmarshal([]byte(cache), &talks)
		if err != nil {
			return nil, err
		}
		println("redis")
		return &talks, nil
	}

	tx, err := t.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	args := []any{id}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	var talk []Talk
	for rows.Next() {
		var r Talk
		err := rows.Scan(&r.Kind, &r.Time, &r.TargetLanguage, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		talk = append(talk, r)
	}

	bytes, err := json.Marshal(talk)
	err = t.RDB.Set(ctx, `talk:user:`+id, bytes, 0).Err()
	if err != nil {
		return nil, err
	}

	return &talk, nil
}
