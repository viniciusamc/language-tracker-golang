package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrPageNumberTooLow  = errors.New("The actual page number is less than the number of pages read")
	ErrPageNumberTooHigh = errors.New("The actual page number exceeds the total number of pages")
)

type BookModel struct {
	DB  *pgxpool.Pool
	RDB *redis.Client
}

type DataBooks struct {
	Books            []Book         `json:"books"`
	BooksHistory     []BooksHistory `json:"booksHistory"`
	BooksLastHistory []BooksHistory `json:"booksLastHistory"`
	Kind             string         `json:"source"`
	DurationBooks    time.Duration  `json:"-"`
	TotalTimeBooks   string         `json:"totalTimeBooks"`
	TotalBooksWords  int64          `json:"totalBooksWords"`
	TotalBooksPages  int64          `json:"totalBooksPages"`
	TotalBooks       int            `json:"totalBooks"`
}

type Book struct {
	ID             string    `json:"id"`
	IDUser         string    `json:"-"`
	Title          string    `json:"title"`
	Description    *string   `json:"description"`
	TargetLanguage string    `json:"target_language"`
	CreatedAt      time.Time `json:"created_at"`
	Kind           string    `json:"source"`
}

type BooksHistory struct {
	ID         int64         `json:"-"`
	IDUser     string        `json:"-"`
	IDBook     string        `json:"id_book"`
	ActualPage int64         `json:"actual_page"`
	TotalPages int64         `json:"total_pages"`
	ReadType   string        `json:"read_type"`
	TotalWords int64         `json:"total_words"`
	Time       string        `json:"time"`
	TimeDiff   string        `json:"time_diff"`
	CreatedAt  time.Time     `json:"created_at"`
	RawTime    time.Duration `json:"-"`
	Kind       string        `json:"source"`
}

func (b BookModel) Insert(user *User, title string, pages string, targetLanguage string, minutesReading int) error {
	query := "INSERT INTO books(id_user, title, target_language) VALUES($1, $2, $3) RETURNING id"

	ctx := context.Background()

	tx, err := b.DB.Begin(ctx)
	if err != nil {
		return err
	}

	var idBook uuid.UUID

	args := []any{user.Id.String(), title, targetLanguage}
	err = tx.QueryRow(ctx, query, args...).Scan(&idBook)
	if err != nil {
		return err
	}

	query = "INSERT INTO books_history(id_user, id_book, actual_page, total_pages, read_type, total_words, time) VALUES($1, $2, $3, $4, $5, $6, $7)"

	totalWords := user.Configs.ReadWordsPerMinute * user.Configs.AverageWordsPerPage
	totalTime := minutesReading

	args = []any{user.Id.String(), idBook, 0, pages, "None", totalWords, ParseMinutes(int32(totalTime))}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	b.RDB.Del(ctx, "books:user:"+user.Id.String())

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (b BookModel) GetByUser(user *User) (*DataBooks, error) {
	query := "SELECT id, title, description, target_language, created_at FROM books WHERE id_user = $1"
	queryHistory := "SELECT id,id_book,actual_page, total_pages, read_type, total_words, created_at, time::interval, time_diff::interval FROM books_history WHERE id_user = $1"

	ctx := context.Background()

	cache, err := b.RDB.Get(ctx, "books:user:"+user.Id.String()).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if err != redis.Nil {
		var data DataBooks
		err := json.Unmarshal([]byte(cache), &data)
		if err != nil {
			return nil, err
		}
		return &data, nil
	}

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
		b.Kind = "Books"

		books = append(books, b)
	}

	rows, err = tx.Query(ctx, queryHistory, user.Id.String())
	if err != nil {
		return nil, err
	}

	var booksHistory []BooksHistory
	data := DataBooks{}
	for rows.Next() {
		b := BooksHistory{}
		var rawTimeString time.Duration
		var rawTimeDiff sql.NullString
		err := rows.Scan(&b.ID, &b.IDBook, &b.ActualPage, &b.TotalPages, &b.ReadType, &b.TotalWords, &b.CreatedAt, &rawTimeString, &rawTimeDiff)
		if err != nil {
			return nil, err
		}

		b.Time = ParseTime(rawTimeString)
		if rawTimeDiff.Valid {
			b.TimeDiff = rawTimeDiff.String
		} else {
			b.TimeDiff = "00:00:00"
		}
		b.Kind = "BooksHistory"

		booksHistory = append(booksHistory, b)
	}

	lastHistoryMap := make(map[string]BooksHistory)

	for _, b := range booksHistory {
		lastHistoryMap[b.IDBook] = b
	}

	for _, a := range lastHistoryMap {
		t, _ := ParseDuration(a.Time)
		data.BooksLastHistory = append(data.BooksLastHistory, a)
		data.DurationBooks += t.Abs()
		data.TotalBooksWords += a.TotalWords
		data.TotalBooksPages += a.ActualPage
	}

	data.Books = books
	data.BooksHistory = booksHistory
	data.TotalTimeBooks = ParseTime(data.DurationBooks)
	data.TotalBooks = len(books)

	if len(books) == 0 {
		data = DataBooks{
			Books:            make([]Book, 1),
			BooksHistory:     make([]BooksHistory, 1),
			BooksLastHistory: make([]BooksHistory, 1),
			TotalTimeBooks:   "00:00:00",
		}
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	err = b.RDB.Set(ctx, "books:user:"+user.Id.String(), bytes, 0).Err()
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func (b BookModel) UpdateBook(user *User, idBook string, readPages int, readType string, minutesReading int) error {
	query := "INSERT INTO books_history(id_user, id_book, actual_page, read_type, total_words, time_diff, time, total_pages) VALUES($1, $2, $3, $4, $5, $6, $7, $8)"
	queryHistory := "SELECT actual_page, total_pages, time::interval FROM books_history WHERE id_user = $1 AND id_book = $2 ORDER BY created_at DESC LIMIT 1"

	ctx := context.Background()

	tx, err := b.DB.Begin(ctx)
	if err != nil {
		return err
	}

	args := []any{user.Id.String(), idBook}

	var actualPage, totalPages int
	var timeBook time.Duration

	err = tx.QueryRow(ctx, queryHistory, args...).Scan(&actualPage, &totalPages, &timeBook)
	if err != nil {
		return err
	}

	if actualPage >= readPages {
		return ErrPageNumberTooLow
	} else if readPages > totalPages {
		return ErrPageNumberTooHigh
	}

	totalWords := user.Configs.AverageWordsPerPage * int32(readPages)
	timeBookInMinutes := timeBook.Minutes()
	totalTime := minutesReading + int(timeBookInMinutes)

	timeDiff := minutesReading

	args = []any{user.Id.String(), idBook, readPages, readType, totalWords, ParseMinutes(int32(timeDiff)), ParseMinutes(int32(totalTime)), totalPages}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	b.RDB.Del(ctx, "books:user:"+user.Id.String())

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (b BookModel) Delete(user *User, idBook string) error {
	query := "DELETE FROM books WHERE id_user = $1 AND id = $2"

	ctx := context.Background()

	tx, err := b.DB.Begin(ctx)
	if err != nil {
		return err
	}

	args := []any{user.Id.String(), idBook}

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	b.RDB.Del(ctx, "books:user:"+user.Id.String())

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}
