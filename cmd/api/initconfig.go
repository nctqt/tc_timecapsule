package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	tc "github.com/nctqt/tc_timecapsule"
	"github.com/nctqt/tc_timecapsule/internal/database"
	"github.com/pressly/goose/v3"
)

type apiConfig struct {
	dbURL    string            // database url
	db       *sql.DB           // db
	queries  *database.Queries // sqlc generated query handler
	mux      *http.ServeMux    // http router
	httpPort string            // server port
	httpHost string            // server host
	logFile  *os.File          // log file
}

func initialConfig() *apiConfig {
	apiCfg := &apiConfig{}

	// log
	logfile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error creating log: %v", err)
	}
	apiCfg.logFile = logfile
	multiWriter := io.MultiWriter(os.Stderr, logfile)
	log.SetOutput(multiWriter)

	// env
	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading env: %v", err)
	}

	// config env vars
	apiCfg.dbURL = os.Getenv("DATABASE_URL")
	if apiCfg.dbURL == "" {
		log.Fatal("DATABASE_URL env variable is required")
	}
	apiCfg.httpPort = os.Getenv("HTTP_PORT")
	if apiCfg.httpPort == "" {
		log.Fatal("HTTP_PORT env variable is required")
	}
	apiCfg.httpHost = os.Getenv("HTTP_HOST")
	if apiCfg.httpHost == "" {
		log.Fatal("HTTP_HOST env variable is required")
	}

	// db
	log.Println("Connecting to PostgreSQL...")
	db, err := sql.Open("pgx", apiCfg.dbURL)
	if err != nil {
		log.Fatalf("Error opening db: %v", err)
	}
	apiCfg.db = db

	// verify connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	// migrations via goose
	goose.SetBaseFS(tc.EmbedMigrations)
	err = goose.SetDialect("postgres")
	if err != nil {
		log.Fatalf("Failed to set goose dialect: %v", err)
	}

	log.Println("Running database migrations...")
	err = goose.Up(db, "sql/schema")
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Database migrations complete!")

	// sqlc queries method access for db interfacing
	apiCfg.queries = database.New(db)
	return apiCfg
}
