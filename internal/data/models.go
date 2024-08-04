package data

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Models struct {
	Users UserModel
	Talks TalkModel
	Medias MediasModel
	Anki AnkiModel
	Vocabulary VocabularyModel
	Book BookModel
}

func NewModel(db *pgxpool.Pool, rdb *redis.Client) Models {
	return Models{
		Users: UserModel{db, rdb},
		Talks: TalkModel{db, rdb},
		Medias: MediasModel{db, rdb},
		Anki: AnkiModel{db, rdb},
		Vocabulary: VocabularyModel{db, rdb},
		Book: BookModel{db, rdb},
	}
}
