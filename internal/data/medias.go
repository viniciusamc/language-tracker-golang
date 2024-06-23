package data

import (
	"context"
	"fmt"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type MediasModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
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
	query := `INSERT INTO medias(id_user, video_id, type, watch_type, target_language, title) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`
	ctx := context.Background()

	tx, err := t.DB.Begin(ctx)
	if err != nil {
		return "", "", err
	}

	videoId, err := ExtractYouTubeVideoID(url)
	if err != nil {
		return "", "", err
	}

	args := []any{userId, videoId, kind, watchType, targetLanguage, "Youtube"}
	var idMedia string

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
