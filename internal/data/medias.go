package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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
	ID             string      `json:"-"`
	IDUser         string      `json:"-"`
	Title          string      `json:"title"`
	VideoID        string      `json:"video_id"`
	Episode        interface{} `json:"episode"`
	Type           string      `json:"type"`
	WatchType      string      `json:"watch_type"`
	Time           string      `json:"time"`
	TargetLanguage string      `json:"target_language"`
	CreatedAt      time.Time   `json:"created_at"`
	TotalWords     int         `json:"total_words"`
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
		return "", fmt.Errorf("invalid YouTube URL")
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

	args := []any{userId, videoId, kind, watchType, targetLanguage, "Youtube", "00:00:00"}
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
			title, video_id, episode, type, watch_type, time, created_at, target_language, 
			SUM(time) OVER (PARTITION BY id_user) as total_time, 
			SUM(total_words) OVER (PARTITION BY id_user) as sum_words 
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
			return Medias{}, err
		}
		fmt.Println("medias cached")
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
		var t string
		err := rows.Scan(&r.Title, &r.VideoID, &r.Episode, &r.Type, &r.WatchType, &t, &r.CreatedAt, &r.TargetLanguage, &totalDuration, &totalWords)
		if err != nil {
			return Medias{}, err
		}
		r.Time = t
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

	err = t.RDB.Set(ctx, "medias:user:"+userId, mediasByte, 24*time.Hour).Err()
	if err != nil {
		return Medias{}, err
	}

	return medias, nil
}
