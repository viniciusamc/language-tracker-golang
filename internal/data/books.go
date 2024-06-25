package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type BookModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
}

type DataBooks struct {
	Books            []Book         `json:"books"`
	BooksHistory     []BooksHistory `json:"booksHistory"`
	BooksLastHistory []BooksHistory `json:"booksLastHistory"`
	TotalTimeBooks   string         `json:"totalTimeBooks"`
	TotalBooksWords  int64          `json:"totalBooksWords"`
	TotalBooksPages  int64          `json:"totalBooksPages"`
}

type Book struct {
	ID             string    `json:"id"`
	IDUser         string    `json:"id_user"`
	Title          string    `json:"title"`
	Description    *string   `json:"description"`
	TargetLanguage string    `json:"target_language"`
	CreatedAt      time.Time `json:"created_at"`
}

type BooksHistory struct {
	ID         int64         `json:"id"`
	IDUser     string        `json:"id_user"`
	IDBook     string        `json:"id_book"`
	ActualPage int64         `json:"actual_page"`
	TotalPages int64         `json:"total_pages"`
	ReadType   string        `json:"read_type"`
	TotalWords int64         `json:"total_words"`
	Time       time.Duration `json:"time"`
	TimeDiff   *string       `json:"time_diff"`
	CreatedAt  time.Time     `json:"created_at"`
}

func (b BookModel) Insert(user *User, title string, description string, pages string, readPages string, readType string, targetLanguage string) error {
	query := "INSERT INTO books(id_user, title, description, target_language) VALUES($1, $2, $3, $4) RETURNING id"

	ctx := context.Background()

	tx, err := b.DB.Begin(ctx)
	if err != nil {
		return err
	}

	var idBook uuid.UUID

	args := []any{user.Id.String(), title, description, targetLanguage}
	err = tx.QueryRow(ctx, query, args...).Scan(&idBook)
	if err != nil {
		return err
	}

	query = "INSERT INTO books_history(id_user, id_book, actual_page, total_pages, read_type, total_words, time) VALUES($1, $2, $3, $4, $5, $6, $7)"

	totalWords := user.Configs.ReadWordsPerMinute * user.Configs.AverageWordsPerPage
	totalTime := totalWords / user.Configs.ReadWordsPerMinute

	args = []any{user.Id.String(), idBook, readPages, pages, readType, totalWords, ParseMinutes(totalTime)}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	b.RDB.Del(ctx, "books:user:"+user.Id.String())

	tx.Commit(ctx)

	return nil
}

func (b BookModel) GetByUser(user *User) (*DataBooks, error) {
	query := "SELECT id, title, description, target_language, created_at FROM books WHERE id_user = $1"
	queryHistory := "SELECT id,id_book,actual_page, total_pages, read_type, total_words, created_at, time FROM books_history WHERE id_user = $1"

	ctx := context.Background()

	tx, err := b.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, query, user.Id.String())
	if err != nil {
		return nil, err
	}

	var books []Book

	for rows.Next() {
		var b Book
		err := rows.Scan(&b.ID, &b.Title, &b.Description, &b.TargetLanguage, &b.CreatedAt)
		if err != nil {
			return nil, err
		}

		books = append(books, b)
	}

	rows, err = tx.Query(ctx, queryHistory, user.Id.String())
	if err != nil {
		return nil, err
	}

	var booksHistory []BooksHistory
	for rows.Next() {
		var b BooksHistory
		err := rows.Scan(&b.ID, &b.IDBook, &b.ActualPage, &b.TotalPages, &b.ReadType, &b.TotalWords, &b.CreatedAt, &b.Time)
		if err != nil {
			return nil, err
		}

		booksHistory = append(booksHistory, b)
	}

	lastHistoryMap := make(map[string]BooksHistory)

	for _, b := range(booksHistory) {
		lastHistoryMap[b.IDBook] = b
	}

	var data DataBooks
	for _, a := range lastHistoryMap{
		fmt.Println(a.IDBook)
		data.BooksLastHistory = append(data.BooksLastHistory, a)
	}


	data.Books = books
	data.BooksHistory = booksHistory

	tx.Commit(ctx)

	return &data, nil
}
