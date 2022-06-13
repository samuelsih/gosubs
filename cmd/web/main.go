package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync"

	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

const port = ":80"

func main() {
	//database
	db := initDB()
	db.Ping()

	//sessions
	sessions := initSessions()

	//loggers
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	//wait group
	wg := sync.WaitGroup{}

	//config
	app := Config{
		Session: sessions,
		DB: db,
		Wg: &wg,
		InfoLog: infoLog,
		ErrorLog: errorLog,
	}

	app.serve()
}

func(app *Config) serve() {
	server := http.Server{
		Addr: port,
		Handler: app.routes(),
	}

	app.InfoLog.Println("Server start...")
	if err := server.ListenAndServe(); err != nil {
		log.Panic(err)
	}


}

func initDB() *sql.DB {
	connection := connectToDB()

	if connection == nil {
		log.Fatal("can't connect to postgres")
	}

	return connection
}

func connectToDB() *sql.DB {
	counts := 0

	dsn := os.Getenv("DSN")

	for {
		conn, err := openDB(dsn)
		if err != nil {
			log.Println("postgres can't ready...", err)
		} else {
			log.Print("connected to postgres")
			return conn
		}

		if counts > 10 {
			return nil
		}

		log.Print("Sleep for 1 second")
		time.Sleep(1 * time.Second)
		counts++
		continue
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initSessions() *scs.SessionManager {
	sess := scs.New()

	sess.Store = redisstore.New( initRedis() )
	sess.Lifetime = 24 * time.Hour
	sess.Cookie.Persist = true
	sess.Cookie.SameSite = http.SameSiteLaxMode
	sess.Cookie.Secure = true

	return sess
}

func initRedis() *redis.Pool {
	return &redis.Pool{
		MaxIdle: 10,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", os.Getenv("REDIS"))
		},
	}
}
