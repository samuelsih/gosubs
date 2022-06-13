package data

import (
	"database/sql"
	"time"
)

//waktu query > 3 second? ada error
const dbTimeout = 3 * time.Second

var db *sql.DB

func New(dbPool *sql.DB) Models {
	db = dbPool

	return Models{
		User: User{},
		Plan: Plan{},
	}
}

type Models struct {
	User User
	Plan Plan
}
