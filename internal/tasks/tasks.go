package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"regexp"
	"strconv"
	"strings"

	youtubetranscript "github.com/dougbarrett/youtube-transcript"
	"github.com/gocolly/colly"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	TypeEmailDelivery = "email:deliver"
	TypeTranscript    = "media:transcript"
)

type EmailDeliveryPayload struct {
	UserID     string
	TemplateID string
}

type TranscriptPayload struct {
	UserId         string
	MediaId        string
	TargetLanguage string
	YoutubeUrl     string
}


func parseISO8601Duration(iso8601 string) (string, error) {
	re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)
	matches := re.FindStringSubmatch(iso8601)

	if matches == nil {
		return "", fmt.Errorf("invalid ISO8601 duration format")
	}

	var hours, minutes, seconds int
	var err error

	if matches[1] != "" {
		hours, err = strconv.Atoi(matches[1])
		if err != nil {
			return "", err
		}
	}
	if matches[2] != "" {
		minutes, err = strconv.Atoi(matches[2])
		if err != nil {
			return "", err
		}
	}
	if matches[3] != "" {
		seconds, err = strconv.Atoi(matches[3])
		if err != nil {
			return "", err
		}
	}

	totalMinutes := hours*60 + minutes
	formattedHours := totalMinutes / 60
	formattedMinutes := totalMinutes % 60

	return fmt.Sprintf("%02d:%02d:%02d", formattedHours, formattedMinutes, seconds), nil
}

func NewMailDeliveryTask(userId string, tmplID string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailDeliveryPayload{UserID: userId, TemplateID: tmplID})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeEmailDelivery, payload), nil
}

func NewTranscriptTask(userId string, media string, youtubeUrl string, targetLanguage string) (*asynq.Task, error) {
	payload, err := json.Marshal(TranscriptPayload{UserId: userId, MediaId: media, YoutubeUrl: youtubeUrl, TargetLanguage: targetLanguage})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeTranscript, payload), nil
}

func HandleMailTask(ctx context.Context, t *asynq.Task) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	log.Printf("Sending Email to User: user_id=%s, template_id=%s", p.UserID, p.TemplateID)

	// auth := smtp.PlainAuth("", os.Getenv("EMAIL_USER"), os.Getenv("EMAIL_API_KEY"), os.Getenv("EMAIL_HOST"))

	to := []string{"vinicius@example.com"}

	msg := []byte("To: kate.doe@example.com\r\n" +
		"Subject: Why aren’t you using Mailtrap yet?\r\n" +
		"\r\n" +
		"Here’s the space for our great sales pitch\r\n")

	err := smtp.SendMail(os.Getenv("EMAIL_HOST")+":"+os.Getenv("EMAIL_PORT"), nil, "john.doe@gmail.com", to, msg)
	if err != nil {
		log.Default().Print(err)
	}
	return nil
}

func HandleTranscriptTask(ctx context.Context, t *asynq.Task, rdb *redis.Client, pool *pgxpool.Pool) error {
	var y TranscriptPayload
	if err := json.Unmarshal(t.Payload(), &y); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	opts := []youtubetranscript.Option{
		youtubetranscript.WithLang(y.TargetLanguage),
	}
	log.Printf("Received task: UserId=%s, YoutubeUrl=%s, TargetLanguage=%s, MediaId=%s",
		y.UserId, y.YoutubeUrl, y.TargetLanguage, y.MediaId)
	transcript, err := youtubetranscript.GetTranscript(ctx, y.YoutubeUrl, opts...)
	if err != nil {
		log.Fatalf("Error fetching transcript: %v", err)
	}

	var title, duration string

	c := colly.NewCollector()

	c.OnHTML("title", func(e *colly.HTMLElement) {
		title = e.Text
	})

	c.OnHTML("meta[itemprop=duration]", func(e *colly.HTMLElement) {
		durationContent := e.Attr("content")
		duration, _ = parseISO8601Duration(durationContent)
		println(duration)
		println(durationContent)
	})
	err = c.Visit("https://www.youtube.com/watch?v=" + y.YoutubeUrl)
	if err != nil {
		log.Fatalf("Failed to visit page: %v", err)
	}

	c.Wait()

	separeted := strings.Split(transcript, " ")

	err = rdb.Del(ctx, "medias:user:"+y.UserId).Err()
	if err != nil {
		log.Fatalf("error deleting cache %v", err)
		return err
	}

	query := `UPDATE medias SET total_words = $1, title = $4, time = $5 WHERE id_user = $2 AND id = $3`
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("error begin pool %v", err)
		return err
	}

	defer tx.Rollback(ctx)

	args := []any{len(separeted), y.UserId, y.MediaId, title, duration}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		log.Fatalf("error exec database %v", err)
		return err
	}

	tx.Commit(ctx)

	return nil

}
