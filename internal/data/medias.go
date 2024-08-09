package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	InvalidUrl = errors.New("Invalid Youtube URL")
)

type MediasModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
}

type Medias struct {
	Videos         []Video `json:"videos"`
	Time           string  `json:"time"`
	TotalVideos    int     `json:"totalVideos"`
	TotalWordCount int     `json:"totalWordCount"`
}

type Video struct {
	ID             string         `json:"id"`
	IDUser         string         `json:"-"`
	Title          string         `json:"title"`
	VideoID        sql.NullString `json:"video_id"`
	Episode        sql.NullString `json:"episode"`
	Type           string         `json:"type"`
	WatchType      string         `json:"watch_type"`
	Time           string         `json:"time"`
	TargetLanguage string         `json:"target_language"`
	CreatedAt      time.Time      `json:"created_at"`
	TotalWords     int            `json:"total_words"`
	Kind           string         `json:"source"`
}

type UpdateV struct {
	IdUser string
	IdMedia string
	VideoId string
	TargetLanguage string
}

func ParseDuration(hms string) (time.Duration, error) {
	parts := strings.Split(hms, ":")
	if len(parts) != 3 {
		return 0, errors.New("invalid duration format")
	}

	hours := parts[0]
	minutes := parts[1]
	seconds := parts[2]

	durationString := hours + "h" + minutes + "m" + seconds + "s"
	return time.ParseDuration(durationString)
}

func ParseTime(time time.Duration) string {
	return fmt.Sprintf("%02d:%02d:%02d", int(time.Hours()), int(time.Minutes())%60, int(time.Seconds())%60)
}

func ExtractYouTubeVideoID(url string) (string, error) {
	re := regexp.MustCompile(`(?:https?:\/\/)?(?:www\.)?(?:youtube\.com\/(?:[^\/\n\s]+\/\S+\/|(?:v|e(?:mbed)?)\/|\S*?[?&]v=)|youtu\.be\/)([a-zA-Z0-9_-]{11})`)
	match := re.FindStringSubmatch(url)
	if len(match) < 2 {
		return "", InvalidUrl
	}
	return match[1], nil
}

func (t MediasModel) Insert(userId string, url string, kind string, watchType string, targetLanguage string) (string, string, error) {
	query := `INSERT INTO medias(id_user, video_id, type, watch_type, target_language, title, time) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`
	ctx := context.Background()

	tx, err := t.DB.Begin(ctx)
	if err != nil {
		return "", "", err
	}

	videoId, err := ExtractYouTubeVideoID(url)
	if err != nil {
		return "", "", err
	}

	args := []any{userId, videoId, kind, watchType, targetLanguage, "Processing Video Information – Please Wait", "00:00:00"}
	var idMedia string

	t.RDB.Del(ctx, `medias:user:`+userId)

	err = tx.QueryRow(ctx, query, args...).Scan(&idMedia)
	if err != nil {
		return "", "", err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return "", "", err
	}

	return idMedia, videoId, nil
}

func (t MediasModel) Get(userId string) (Medias, error) {
	query := `
		SELECT 
			id, title, video_id, episode, type, watch_type, time::interval, created_at, target_language, 
			SUM(time) OVER (PARTITION BY id_user) as total_time, 
			SUM(total_words) OVER (PARTITION BY id_user) as sum_words,
			total_words
		FROM medias 
		WHERE id_user = $1`

	ctx := context.Background()

	cache, err := t.RDB.Get(ctx, "medias:user:"+userId).Result()
	if err != nil && err != redis.Nil {
		return Medias{}, err
	}

	if err != redis.Nil {
		var medias Medias
		err := json.Unmarshal([]byte(cache), &medias)
		if err != nil {
			t.RDB.Del(ctx, `medias:user:`+userId)
			return Medias{}, err
		}
		return medias, nil
	}

	rows, err := t.DB.Query(ctx, query, userId)
	if err != nil {
		return Medias{}, err
	}
	defer rows.Close()

	var videos []Video
	var totalDuration time.Duration
	var totalWords int
	for rows.Next() {
		var r Video
		var t time.Duration
		err := rows.Scan(&r.ID, &r.Title, &r.VideoID, &r.Episode, &r.Type, &r.WatchType, &t, &r.CreatedAt, &r.TargetLanguage, &totalDuration, &totalWords, &r.TotalWords)
		if err != nil {
			return Medias{}, err
		}
		r.Time = ParseTime(t)
		r.Kind = "Medias"
		videos = append(videos, r)
	}
	if err := rows.Err(); err != nil {
		return Medias{}, err
	}

	var medias Medias
	medias.TotalVideos = len(videos)
	medias.Videos = videos
	medias.Time = ParseTime(totalDuration)
	medias.TotalWordCount = totalWords

	if len(videos) == 0 {
		medias.Videos = make([]Video, 1)
	}

	mediasByte, err := json.Marshal(medias)
	if err != nil {
		return Medias{}, err
	}

	err = t.RDB.Set(ctx, "medias:user:"+userId, mediasByte, 10*time.Minute).Err()
	if err != nil {
		return Medias{}, err
	}

	return medias, nil
}

func (t MediasModel) Delete(user *User, id string) (string, string, error) {
	query := "DELETE FROM medias WHERE id_user = $1 AND id = $2 RETURNING video_id, target_language"

	tx, err := t.DB.Begin(context.Background())
	if err != nil {
		return "", "", err
	}

	t.RDB.Del(context.Background(), "medias:user:"+user.Id.String())

	var videoId, targetLanguage string

	args := []any{user.Id.String(), id}
	err = tx.QueryRow(context.Background(), query, args...).Scan(&videoId, &targetLanguage)

	if err != nil {
		return "", "", err
	}

	err = tx.Commit(context.Background())
	if err != nil {
		return "", "", err
	}

	return videoId, targetLanguage, nil
}

func (t MediasModel) UpdateAll(user *User) ([]UpdateV, error) {
	query := "SELECT id, id_user, video_id, target_language from medias WHERE video_id IS NOT NULL AND total_words > 0"

	rows, err := t.DB.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	var list []UpdateV

	for rows.Next() {
		var v UpdateV
		err := rows.Scan(&v.IdMedia, &v.IdUser, &v.VideoId, &v.TargetLanguage)
		if err != nil {
			fmt.Println(err.Error())
			return nil, err
		}

		list = append(list, v)
	}

	return list, nil
}
