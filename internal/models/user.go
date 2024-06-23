package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id             uuid.UUID 
	Username       string
	Email          string
	Password       string
	Configs        []string
	Email_token    uuid.UUID
	Email_verified bool
	Created_at     time.Time
	Updated_at     time.Time
}
