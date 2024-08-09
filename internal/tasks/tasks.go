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
	TypeDeleteWords              = "media:delete"
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

type DeleteWordsPayload struct {
	UserId         string
	TargetLanguage string
	YoutubeId      string
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

	return asynq.NewTask(TypeTranscript, payload, asynq.MaxRetry(2)), nil
}

func NewDeleteWordsTask(userId string, youtubeId string, targetLanguage string) (*asynq.Task, error) {
	payload, err := json.Marshal(DeleteWordsPayload{UserId: userId, YoutubeId: youtubeId, TargetLanguage: targetLanguage})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeDeleteWords, payload), nil
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

		subject := "Subject: Verify Your Language Tracker account\n"
		body := "Welcome to Language Tracker, <a href=http://localhost:5173/token/\"" + p.Token + "\">click here to verify your account.</a>"
		msg := []byte(subject + "\n" + body)

		err := smtp.SendMail(smtpHost+":"+smtpPort, nil, from, to, msg)

		if err != nil {
			fmt.Print(err.Error())
			return err
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
	fmt.Println(y.YoutubeUrl)
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
		log.Fatalf("Failed to visit page: %v", err.Error())
	}

	c.Wait()

	separeted := strings.Split(transcript, " ")

	insertWords := "INSERT INTO words(word) VALUES($1) RETURNING id, word"
	searchWords := "SELECT id, word FROM words WHERE word = $1"
	insertWordsAmount := "INSERT INTO aux_words_amount(id_user, word, amount, language) VALUES ($1, $2, $3, $4) ON CONFLICT (word, id_user) DO UPDATE SET amount = aux_words_amount.amount + EXCLUDED.amount"

	txWords, err := pool.Begin(ctx)
	if err != nil {
		fmt.Print(err.Error())
		return err
	}

	wordWithoutDuplicates := make(map[string]int)

	totalWords := 0
	re := regexp.MustCompile(`[\P{L}]+`) // removing digits, whitespaces, symbols and punctuations
	for _, rawWord := range separeted {
		word := re.ReplaceAllString(rawWord, "")

		if len(word) <= 2 {
			continue
		}

		wordWithoutDuplicates[strings.ToLower(word)] += 1
		totalWords++
	}

	for word, amount := range wordWithoutDuplicates {
		var id int
		var wordS string

		errS := txWords.QueryRow(context.Background(), searchWords, word).Scan(&id, &wordS)

		if errS != nil && errS.Error() != `no rows in result set` {
			fmt.Println(errS.Error())
			return errS
		}

		if errS != nil && errS.Error() == `no rows in result set` {
			errI := txWords.QueryRow(context.Background(), insertWords, word).Scan(&id, &wordS)

			if errI != nil && errI.Error() != `ERROR: duplicate key value violates unique constraint "words_word_key" (SQLSTATE 23505)` {
				fmt.Println(errI.Error())
				return errI
			}
		}

		args := []any{y.UserId, &id, amount, y.TargetLanguage}
		_, errL := txWords.Exec(context.Background(), insertWordsAmount, args...)
		if errL != nil {
			fmt.Println(errL)
			return errL
		}
	}

	err = txWords.Commit(context.Background())
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	tx, err := pool.Begin(context.Background())

	query := `UPDATE medias SET total_words = $1, title = $4, time = $5 WHERE id_user = $2 AND id = $3`
	if err != nil {
		log.Fatalf("error begin pool %v", err.Error())
		return err
	}

	defer tx.Rollback(ctx)

	args := []any{len(separeted) - 1, y.UserId, y.MediaId, title, duration}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		log.Fatalf("error exec database %v", err.Error())
		return err
	}

	err = rdb.Del(ctx, "medias:user:"+y.UserId).Err()
	if err != nil {
		log.Fatalf("error deleting cache %v", err.Error())
		return err
	}

	tx.Commit(ctx)

	return nil

}

func HandleDeleteTranscriptTask(ctx context.Context, t *asynq.Task, rdb *redis.Client, pool *pgxpool.Pool) error {
	var y DeleteWordsPayload
	if err := json.Unmarshal(t.Payload(), &y); err != nil {
		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}
	opts := []youtubetranscript.Option{
		youtubetranscript.WithLang(y.TargetLanguage),
	}

	transcript, err := youtubetranscript.GetTranscript(ctx, y.YoutubeId, opts...)

	if err != nil && !strings.Contains(err.Error(), "no transcript found") {
		return err
	}

	separeted := strings.Split(transcript, " ")

	deleteWords := "UPDATE SET amount = aux_words.amount - $1 FROM aux_words_amount"

	txWords, err := pool.Begin(ctx)
	if err != nil {
		fmt.Print(err.Error())
		return err
	}

	wordWithoutDuplicates := make(map[string]int)

	totalWords := 0
	re := regexp.MustCompile(`[\P{L}]+`) // removing digits, whitespaces, symbols and punctuations
	for _, rawWord := range separeted {
		word := re.ReplaceAllString(rawWord, "")

		if len(word) <= 2 {
			continue
		}

		wordWithoutDuplicates[strings.ToLower(word)] += 1
		totalWords++
	}

	for word := range wordWithoutDuplicates {
		_, err := txWords.Exec(context.Background(), deleteWords, word)

		if err != nil {
			fmt.Println(err.Error())
			return err
		}

	}

	err = txWords.Commit(context.Background())
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}
