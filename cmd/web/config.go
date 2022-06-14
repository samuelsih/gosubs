package main

import (
	"database/sql"
	"log"
	"sync"

	"gosubs/data"

	"github.com/alexedwards/scs/v2"
)

type Config struct {
	Session  *scs.SessionManager
	DB       *sql.DB
	InfoLog  *log.Logger
	ErrorLog *log.Logger
	Wg       *sync.WaitGroup
	Models   data.Models
	Mailer   Mail
}
