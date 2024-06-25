package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type VocabularyModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
}

type DataVocabulary struct {
	Vocabulary []Vocabulary `json:"vocabulary"`
	Average    int32        `json:"average"`
}

type Vocabulary struct {
	ID                  string    `json:"id"`
	Vocabulary          int       `json:"vocabulary"`
	DifferenceLastMonth int64     `json:"difference_last_month"`
	URL                 *string   `json:"url"`
	TargetLanguage      string    `json:"target_language"`
	Date                time.Time `json:"date"`
}

func (v VocabularyModel) Insert(user string, vocabulary int32, targetLanguage string) error {
	query := "INSERT INTO vocabulary(id_user, vocabulary, target_language, diff_last) VALUES ($1, $2, $3, $4)"
	lastDiff := "SELECT vocabulary FROM vocabulary WHERE id_user = $1 ORDER BY created_at ASC"

	ctx := context.Background()

	tx, err := v.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)
	var lastVocabulary int

	err = tx.QueryRow(ctx, lastDiff, user).Scan(&lastVocabulary)

	args := []any{user, vocabulary, targetLanguage, vocabulary - int32(lastVocabulary)}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	v.RDB.Del(ctx, "vocabulary:user:"+user)
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (v VocabularyModel) GetByUser(user string) (*DataVocabulary, error) {
	query := "SELECT vocabulary, diff_last, target_language, created_at, AVG(diff_last) OVER (PARTITION BY diff_last) FROM vocabulary WHERE id_user = $1"

	ctx := context.Background()

	tx, err := v.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, query, user)
	if err != nil {
		return nil, err
	}

	var DataVocabulary DataVocabulary
	var vocabulary []Vocabulary
	for rows.Next() {
		var v Vocabulary
		err := rows.Scan(&v.Vocabulary, &v.DifferenceLastMonth, &v.TargetLanguage, &v.Date, &DataVocabulary.Average)
		if err != nil {
			return nil, err
		}
		vocabulary = append(vocabulary, v)
	}

	DataVocabulary.Vocabulary = vocabulary

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &DataVocabulary, nil
}
