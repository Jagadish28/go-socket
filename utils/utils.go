package utils

import (
	"database/sql"
	"log"
	"os"
	"os/exec"
	"sensibull/config"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

var tokensToSubscribe = []int{}

func DataSetup(config *config.AppConfig, l *log.Logger) {
	createDB(config, l)
	connectionCheck(config, l)
	runMigrations(l)
}

// createDB checks whether DB exists or will create a new one
func createDB(config *config.AppConfig, l *log.Logger) {
	var cstr = config.DataBaseCheckConnectionString

	db, err := sql.Open(config.DatabaseHost, cstr)
	if err != nil {
		l.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		l.Fatal(err)
	}

	rows, err := db.Query("SELECT 1 FROM pg_database WHERE datname = 'sensibull'")

	if err != nil {
		l.Fatal(err)
	}
	defer rows.Close()

	if rows.Next() {
		// Database already exists
		l.Println("Database already exists")
	} else {
		// Create the database
		_, err = db.Exec("CREATE DATABASE sensibull")
		if err != nil {
			l.Fatal(err)
		}
		l.Println("Database created")
	}
}

func connectionCheck(config *config.AppConfig, l *log.Logger) {
	// Open a connection to the database
	db, err := sql.Open(config.DatabaseHost, config.DataBaseConnectionString)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Attempt to ping the database to verify the connection
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	l.Println("Successfully connected to the PostgreSQL database!")
}

// runMigrations Soda fizz migrations, this will create a 'stocks' table , the scripts under migration folder will run
func runMigrations(l *log.Logger) {
	cmd := exec.Command("soda", "migrate", "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		l.Fatalf("Failed to execute migrations: %v", err)
	}
}

// stockRefresh clears current stocks in stockt table, fetch fresh underlyings and its derivatives and get the list of new tokens to subscribe
func StockRefresh(config *config.AppConfig, conn *websocket.Conn, l *log.Logger) {
	subCh := make(chan []int)
	go LoadInitialStocks(config, subCh, l)
	go func() {
		for {
			select {
			case tokensToSubscribe = <-subCh:
				err := SubscribeToWebsocket(config, conn, tokensToSubscribe, l)
				if err != nil {
					l.Println("Error while subscribing:", err)
				}
			default:
				time.Sleep(1 * time.Second) // to avoid busy-waiting.
			}
		}
	}()
}
