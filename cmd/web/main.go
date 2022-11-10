package main

import (
	"database/sql"
	"encoding/gob"
	"final-project/data"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/gomodule/redigo/redis"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var webport = "80"

func main() {
	// connected to database
	db := initDB()

	// create sessions
	session := initSession()

	// logger (Kalau di production di tulis ke file)
	Infolog := log.New(os.Stdout, "INFO\t ", log.Ldate|log.Ltime)
	Errorlog := log.New(os.Stdout, "Error\t ", log.Ldate|log.Ltime|log.Lshortfile)

	// create channel

	// create waitgroup
	wg := sync.WaitGroup{}

	// setup application config
	app := Config{
		Session:      session,
		DB:           db,
		Wait:         &wg,
		InfoLog:      Infolog,
		ErrorLog:     Errorlog,
		Models:       data.New(db),
		ErrorChan:    make(chan error),
		ErrorChanDne: make(chan bool),
	}

	// setup mail
	app.Mailer = app.CreateMail()
	go app.ListenForMail()

	// Listen For Signal
	go app.ListenForShutdown()

	// listen for error channel
	go app.ListenForError()

	// listen on server
	app.serve()

}
func (app *Config) ListenForError() {
	for {
		select {
		case err := <-app.ErrorChan:
			app.ErrorLog.Println(err)
		case <-app.ErrorChanDne:
			return
		}
	}
}

func (app *Config) serve() {
	// Start http server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", webport),
		Handler: app.routes(),
	}
	app.InfoLog.Println("Starting web Server")

	err := srv.ListenAndServe()
	if err != nil {
		log.Panic(err)
	}
}

func initDB() *sql.DB {
	conn := conncetToDB()
	if conn == nil {
		log.Println("error connect to db")
	}
	return conn
}

func conncetToDB() *sql.DB {
	count := 0
	// dsn := os.Getenv("DSN")
	for {
		connection, err := openDB("host=localhost port=54321 user=postgres password=password dbname=concurrency sslmode=disable timezone=UTC connect_timeout=5")
		if err != nil {
			log.Println("Postgres is not ready")
		} else {
			log.Println("connect to database")
			return connection
		}

		if count > 10 {
			return nil
		}
		log.Println("Backing of 1 time")
		time.Sleep(1 * time.Second)
		count++
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

func initSession() *scs.SessionManager {
	gob.Register(data.User{})

	// setup session
	session := scs.New()
	session.Store = redisstore.New(initRedis())
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = true

	return session

}
func initRedis() *redis.Pool {
	redispool := &redis.Pool{
		MaxIdle: 10,
		Dial: func() (redis.Conn, error) {

			return redis.Dial("tcp", "127.0.0.1:6379")
			// return redis.Dial("tcp", os.Getenv("REDIS"))
		},
	}
	return redispool
}

func (app *Config) ListenForShutdown() {
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	app.Shudown()
	os.Exit(0)
}
func (app *Config) Shudown() {
	// Clean Any task
	app.InfoLog.Println("Akan Membersihkan Task")

	// block until waitgroup is empty

	app.Wait.Wait()

	app.Mailer.DoneChan <- true
	app.ErrorChanDne <- true

	close(app.Mailer.DoneChan)
	close(app.Mailer.ErrorChan)
	close(app.Mailer.MailerChan)
	close(app.ErrorChan)
	close(app.ErrorChanDne)

	app.InfoLog.Println("Closing Channel And Closing Application")
}

func (app *Config) CreateMail() Mail {
	// create channel
	errorChan := make(chan error)
	mailerChan := make(chan Message, 100)
	mailerDoneChan := make(chan bool)

	m := Mail{
		Domain:      "localhost",
		Host:        "localhost",
		Port:        1025,
		Encryption:  "none",
		FromAddress: "test@example.com",
		FromName:    "test",
		Wait:        app.Wait,
		MailerChan:  mailerChan,
		ErrorChan:   errorChan,
		DoneChan:    mailerDoneChan,
	}
	return m
}
