package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type TalkModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
}

type DataOutput struct {
	Output          []Output     `json:"output"`
	OutputTotalTime string       `json:"outputTotalTime"`
	AverageTime     string       `json:"averageTime"`
	OutputStreak    OutputStreak `json:"outputStreak"`
}

type Output struct {
	ID             string    `json:"-"`
	IDUser         string    `json:"-"`
	Kind           string    `json:"type"`
	Time           string    `json:"time"`
	Summarize      string    `json:"summarize"`
	TargetLanguage string    `json:"target_language"`
	CreatedAt      time.Time `json:"created_at"`
}

type OutputStreak struct {
	LongestStreak int64 `json:"longestStreak"`
	CurrentStreak int64 `json:"currentStreak"`
}

func FormatTime(t time.Duration) string {
	d := time.Duration(t)
	return fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}

func ParseMinutes(s int32) string {
	var min time.Time
	min = min.Add(time.Duration(s) * time.Minute)

	return min.Format("15:04:05")
}

func FormatMinutesToDuration(t string) string {
	return ""
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

func (t TalkModel) GetByUser(id string) (DataOutput, error) {
	query := `SELECT type, time, target_language, created_at, AVG(time::interval) OVER (PARTITION BY time) as avg_time, SUM(time::interval) OVER (PARTITION BY time) AS sum_time FROM output WHERE id_user = $1 ORDER BY created_at ASC
	`
	// ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// defer cancel()

	cache, err := t.RDB.Get(context.Background(), `talk:user:`+id).Result()
	if err != nil && err != redis.Nil {
		return DataOutput{}, err
	}

	if err != redis.Nil {
		var output DataOutput
		err := json.Unmarshal([]byte(cache), &output)
		if err != nil {
			return DataOutput{}, err
		}
		println("talk cached")
		return output, nil
	}

	tx, err := t.DB.Begin(context.Background())
	if err != nil {
		return DataOutput{}, err
	}

	defer tx.Rollback(context.Background())

	args := []any{id}

	rows, err := tx.Query(context.Background(), query, args...)
	if err != nil {
		return DataOutput{}, err
	}

	var talk []Output
	var avgTime, totalTime time.Duration
	for rows.Next() {
		var r Output
		err := rows.Scan(&r.Kind, &r.Time, &r.TargetLanguage, &r.CreatedAt, &avgTime, &totalTime)
		if err != nil {
			return DataOutput{}, err
		}
		talk = append(talk, r)
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return DataOutput{}, err
	}

	var output DataOutput

	output.Output = talk

	output.OutputTotalTime = FormatTime(totalTime)
	output.AverageTime = FormatTime(avgTime)

	if len(talk) == 0 {
		output.Output = make([]Output, 1)
	}

	var count = 1
	var bigStreak = 1
	for i := 1; i < len(talk); i++ {
		splited := strings.Split(talk[i].CreatedAt.String(), " ")[0]
		splitb := strings.Split(talk[i-1].CreatedAt.String(), " ")[0]

		splitedP1, _ := time.Parse("2006-01-02", splited)
		splitbP1, _ := time.Parse("2006-01-02", splitb)

		if splitedP1.Sub(splitbP1) == 24*time.Hour {
			count++
		} else {
			if splitb != splited {
				count = 1
			}
		}

		if count > bigStreak {
			bigStreak = count
		}
	}

	output.OutputStreak.CurrentStreak = int64(count)
	output.OutputStreak.LongestStreak = int64(bigStreak)

	bytes, err := json.Marshal(output)
	if err != nil {
		return DataOutput{}, err
	}

	err = t.RDB.Set(context.Background(), `talk:user:`+id, bytes, 0).Err()
	if err != nil {
		return DataOutput{}, err
	}

	return output, nil
}
