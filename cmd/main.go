package main

import (
	"context"
	"language-tracker/internal/data"
	"language-tracker/internal/jsonlog"
	"language-tracker/internal/tasks"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/unrolled/render"
)

type config struct {
	port int
	env  struct {
		Environment string
		JWT_KEY     string
	}
	redis struct {
		port string
		host string
	}
}

type application struct {
	render *render.Render
	log    *jsonlog.Logger
	models data.Models
	queue  *asynq.Client
	config *config
}

func init() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Panic(err)
	}

	var configLoaded config

	configLoaded.env.JWT_KEY = "asda"
	configLoaded.env.Environment = os.Getenv("ENVIRONMENT")

	render := render.New()
	logger := jsonlog.NewLogger(os.Stdout, jsonlog.LevelInfo)

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		Username: os.Getenv("REDIS_USER"),
		DB:       0,
	})
	defer client.Close()

	err = pool.Ping(context.Background())
	if err != nil {
		log.Panic(err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		Username: os.Getenv("REDIS_USER"),
		DB:       0,
	})

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
			Password: os.Getenv("REDIS_PASSWORD"),
			Username: os.Getenv("REDIS_USER"),
		},
		asynq.Config{
			Concurrency: 2,
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeEmailDelivery, tasks.HandleMailTask)
	mux.HandleFunc(tasks.TypeTranscript, func(ctx context.Context, t *asynq.Task) error {
		return tasks.HandleTranscriptTask(ctx, t, rdb, pool)
	})

	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("could not run server: %v", err.Error())
		}
	}()

	app := &application{
		render: render,
		log:    logger,
		models: data.NewModel(pool, rdb),
		queue:  client,
		config: &configLoaded,
	}

	logger.PrintInfo("running on :" + os.Getenv("PORT"), nil)

	server := http.Server{
		Addr:         ":" + os.Getenv("PORT"),
		ErrorLog:     log.New(logger, "", 0),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      app.routes(),
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}
