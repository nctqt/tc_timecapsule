package main

import (
	"database/sql"
	"io"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	tc "github.com/nctqt/tc_timecapsule"
	"github.com/pressly/goose/v3"
)

func main() {
	// log
	logfile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logfile.Close()
	multiWriter := io.MultiWriter(os.Stderr, logfile)
	log.SetOutput(multiWriter)

	// env
	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading env: %v", err)
	}

	// config struct
	apiCfg := &apiConfig{}
	apiCfg.platform = os.Getenv("PLATFORM")
	if apiCfg.platform == "" {
		log.Fatal("PLATFORM env variable is required")
	}
	apiCfg.dbURL = os.Getenv("DATABASE_URL")
	if apiCfg.dbURL == "" {
		log.Fatal("DATABASE_URL env variable is required")
	}

	// db
	log.Println("Connecting to PostgreSQL...")
	db, err := sql.Open("pgx", apiCfg.dbURL)
	if err != nil {
		log.Fatalf("Error opening db: %v", err)
	}
	defer db.Close()

	// migrations
	goose.SetBaseFS(tc.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set goose dialect: %v", err)
	}

	log.Println("Running database migrations...")
	if err := goose.Up(db, "sql/schema"); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Database migrations complete!")

}
