package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"language-tracker/internal/jsonlog"
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
	"github.com/resend/resend-go/v2"
)

const (
	TypeEmailDelivery            = "email:deliver"
	TypeTranscript               = "media:transcript"
	TypeRecoveryPasswordDelivery = "emailPassword:deliver"
)

type EmailDeliveryPayload struct {
	UserID     string
	TemplateID string
	UserEmail  string
	Token      string
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

func NewMailDeliveryTask(userId string, tmplID string, userEmail string, token string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailDeliveryPayload{UserID: userId, TemplateID: tmplID, UserEmail: userEmail, Token: token})
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

func NewRecoveryPasswordTask(userId string, tmplID string, userEmail string, token string) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailDeliveryPayload{UserID: userId, TemplateID: tmplID, UserEmail: userEmail, Token: token})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeRecoveryPasswordDelivery, payload), nil
}

func handleRecoveryPasswordTask(ctx context.Context, t *asynq.Task) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return nil
}

func HandleMailTask(ctx context.Context, t *asynq.Task) error {
	var p EmailDeliveryPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	env := os.Getenv("ENVIRONMENT")

	if env == "production" {
		apiKey := os.Getenv("EMAIL_API_KEY")

		client := resend.NewClient(apiKey)

		params := &resend.SendEmailRequest{
			From:    "Language Tracker <languagetracker@languagetracker.shop>",
			To:      []string{p.UserEmail},
			Html:    "Welcome to Language Tracker, <a href=https://llt-web.vercel.app/token/\"" + p.Token + "\">click here to verify your account.</a>",
			Subject: "Verify Your Language Tracker account",
		}

		_, err := client.Emails.Send(params)
		if err != nil {
			return err
		}

		return nil

	} else {
		from := "languagetracker@languagetracker.com"
		to := []string{p.UserEmail}

		smtpHost := "127.0.0.1"
		smtpPort := "1025"
		auth := smtp.PlainAuth("", from, "", smtpHost)

		subject := "Subject: Verify Your Language Tracker account\n"
		body := "Welcome to Language Tracker, <a href=http://localhost:5173/token/\"" + p.Token + "\">click here to verify your account.</a>"
		msg := []byte(subject + "\n" + body)

		err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, msg)
		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}

		log.Println("Email sent to", p.UserEmail)
		log := jsonlog.NewLogger(os.Stdout, jsonlog.LevelInfo)

		log.PrintInfo("email sent to "+p.UserEmail, nil)

		return nil
	}
}

func HandleTranscriptTask(ctx context.Context, t *asynq.Task, rdb *redis.Client, pool *pgxpool.Pool) error {
	var y TranscriptPayload
	if err := json.Unmarshal(t.Payload(), &y); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	opts := []youtubetranscript.Option{
		youtubetranscript.WithLang(y.TargetLanguage),
	}
	transcript, err := youtubetranscript.GetTranscript(ctx, y.YoutubeUrl, opts...)

	if err != nil && !strings.Contains(err.Error(), "no transcript found") {
		return err
	}

	var title, duration string

	c := colly.NewCollector()

	c.OnHTML("title", func(e *colly.HTMLElement) {
		title = e.Text
	})

	c.OnHTML("meta[itemprop=duration]", func(e *colly.HTMLElement) {
		durationContent := e.Attr("content")
		duration, _ = parseISO8601Duration(durationContent)
	})
	err = c.Visit("https://www.youtube.com/watch?v=" + y.YoutubeUrl)
	if err != nil {
		log.Fatalf("Failed to visit page: %v", err)
	}

	c.Wait()

	separeted := strings.Split(transcript, " ")

	query := `UPDATE medias SET total_words = $1, title = $4, time = $5 WHERE id_user = $2 AND id = $3`
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("error begin pool %v", err)
		return err
	}

	defer tx.Rollback(ctx)

	args := []any{len(separeted) - 1, y.UserId, y.MediaId, title, duration}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		log.Fatalf("error exec database %v", err)
		return err
	}

	err = rdb.Del(ctx, "medias:user:"+y.UserId).Err()
	if err != nil {
		log.Fatalf("error deleting cache %v", err)
		return err
	}

	tx.Commit(ctx)

	return nil

}
