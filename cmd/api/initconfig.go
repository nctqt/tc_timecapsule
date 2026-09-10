package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	tc "github.com/nctqt/casechronicle"
	"github.com/nctqt/casechronicle/internal/database"
	"github.com/nctqt/casechronicle/internal/openrouter"
	"github.com/nctqt/casechronicle/internal/transcript"
	"github.com/nctqt/casechronicle/internal/worker"
	"github.com/nctqt/casechronicle/internal/youtube"
	"github.com/pressly/goose/v3"
)

type apiConfig struct {
	dbURL         string             // database url
	db            *sql.DB            // db
	queries       *database.Queries  // sqlc generated query handler
	mux           *http.ServeMux     // http router
	httpPort      string             // server port
	httpHost      string             // server host
	logFile       *os.File           // log file
	openrouter    *openrouter.Client // openrouter client
	openrouterKey string             // openrouter api key
	yt            *youtube.Client    // yt client
	ytKey         string             // api key
	wp            *worker.WorkerPool // pool of bg workers
	jwtSecret     string             // jwt key
	transcript    *transcript.Client // transcript client
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
	_ = godotenv.Load(".env")

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
	apiCfg.ytKey = os.Getenv("YT_API_KEY")
	if apiCfg.ytKey == "" {
		log.Fatal("YT_API_KEY environment variable is required")
	}
	apiCfg.openrouterKey = os.Getenv("OPENROUTER_API_KEY")
	if apiCfg.openrouterKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}
	apiCfg.jwtSecret = os.Getenv("JWT_SECRET")
	if apiCfg.jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// yt client
	apiCfg.yt, err = youtube.NewClient(apiCfg.ytKey)
	if err != nil {
		log.Fatalf("Error creating youtube client: %v", err)
	}

	// openrouter client
	apiCfg.openrouter, err = openrouter.NewClient(apiCfg.openrouterKey)
	if err != nil {
		log.Fatalf("Error creating openrouter client: %v", err)
	}

	// transcript client
	apiCfg.transcript = transcript.NewClient()

	// db
	log.Println("Connecting to PostgreSQL...")
	db, err := sql.Open("pgx", apiCfg.dbURL)
	if err != nil {
		log.Fatalf("Error opening db: %v", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
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
