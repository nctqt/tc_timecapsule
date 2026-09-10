package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/nctqt/casechronicle/internal/database"
	"github.com/nctqt/casechronicle/internal/openrouter"
	"github.com/nctqt/casechronicle/internal/transcript"
	"github.com/nctqt/casechronicle/internal/worker"
	"github.com/nctqt/casechronicle/internal/youtube"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/pressly/goose/v3"
)

// Standard JSON response shapes matching the api responses
type SignupResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type VideoResponse struct {
	ID             uuid.UUID `json:"id"`
	YoutubeVideoID string    `json:"youtube_video_id"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
}

type TimelineItem struct {
	MilestoneID uuid.UUID       `json:"milestone_id"`
	Title       string          `json:"title"`
	Videos      []VideoResponse `json:"videos"`
}

func setupTestApiConfig(t *testing.T) *apiConfig {
	t.Helper()

	// create Docker pool connection
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not connect to Docker: %s", err)
	}

	// spin up fresh postgres container
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16-alpine",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=testuser",
			"POSTGRES_DB=casechronicle_test",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true // automatically purge container when stopped
	})
	if err != nil {
		t.Fatalf("Could not start container: %s", err)
	}

	// auto-cleanup container when test completes
	t.Cleanup(func() {
		if err := pool.Purge(resource); err != nil {
			log.Printf("Could not purge resource: %s", err)
		}
	})

	var db *sql.DB
	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseURL := fmt.Sprintf("postgres://testuser:secret@%s/casechronicle_test?sslmode=disable", hostAndPort)

	// inline func to open the db
	openDatabase := func() error {
		var err error
		db, err = sql.Open("pgx", databaseURL)
		if err != nil {
			return err
		}
		return db.Ping()
	}

	// pass it to the retry loop
	pool.MaxWait = 10 * time.Second
	err = pool.Retry(openDatabase)
	if err != nil {
		t.Fatalf("Could not connect to Docker Postgres: %s", err)
	}

	// run database migrations on fresh container using goose
	runTestMigrations(t, db)

	// initialize clients via env
	_ = godotenv.Load(".env_test")
	ytKey := os.Getenv("YT_API_KEY")
	openrouterKey := os.Getenv("OPENROUTER_API_KEY")

	queries := database.New(db)
	transcriptClient := transcript.NewClient()
	youtubeClient, _ := youtube.NewClient(ytKey)
	openRouterClient, _ := openrouter.NewClient(openrouterKey)

	// set cfg
	cfg := &apiConfig{
		queries:    queries,
		openrouter: openRouterClient,
		transcript: transcriptClient,
		yt:         youtubeClient,
		jwtSecret:  "test-secret-12345",
		mux:        http.NewServeMux(),
	}

	// instantiate Worker Pool BEFORE registering endpoints
	wp := worker.NewWorkerPool(2, cfg.processVideoEnrichment)
	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx, 2)
	t.Cleanup(func() {
		cancel()
		wp.Stop()
	})

	// attach worker pool to apiConfig
	cfg.wp = wp

	// register Endpoints LAST (so handlers have non-nil cfg.wp)
	registerEndpoints(cfg)

	return cfg
}

func runTestMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("Failed to set goose dialect: %v", err)
	}

	// run all 'up' migrations from your migrations directory
	if err := goose.Up(db, "../../sql/schema"); err != nil { // Adjust path to your migrations folder
		t.Fatalf("Failed to run goose test migrations: %v", err)
	}
}

func TestFullIntegrationPipeline(t *testing.T) {
	// setup test server & dependencies
	cfg := setupTestApiConfig(t)
	server := httptest.NewServer(cfg.mux)
	defer server.Close()

	client := server.Client()

	// Shared state across test sub-steps
	var authToken string
	var createdVideoID string

	// -------------------------------------------------------------------------
	// STEP 1: USER AUTHENTICATION (Signup & Login)
	// -------------------------------------------------------------------------
	t.Run("1. User Signup & Login", func(t *testing.T) {
		email := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
		password := "SecurePassword123!"

		// signup
		signupBody, _ := json.Marshal(map[string]string{
			"email":    email,
			"password": password,
		})
		resp, err := client.Post(server.URL+"/api/v1/users/register", "application/json", bytes.NewBuffer(signupBody))
		if err != nil {
			t.Fatalf("Signup failed request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 201/200 on signup, got %d", resp.StatusCode)
		}
		resp.Body.Close()

		// login to obtain JWT
		loginBody, _ := json.Marshal(map[string]string{
			"email":    email,
			"password": password,
		})
		resp, err = client.Post(server.URL+"/api/v1/users/login", "application/json", bytes.NewBuffer(loginBody))
		if err != nil {
			t.Fatalf("Login failed request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200 on login, got %d", resp.StatusCode)
		}

		var loginData LoginResponse
		json.NewDecoder(resp.Body).Decode(&loginData)
		resp.Body.Close()

		if loginData.Token == "" {
			t.Fatal("Expected JWT token in login response, got empty string")
		}

		// store token in parent scope for subsequent steps
		authToken = loginData.Token
		t.Logf("✓ Auth verified: Received JWT Token (%s...)", authToken[:15])
	})

	// helper closure to create authenticated HTTP requests
	authRequest := func(method, url string, body []byte) (*http.Request, error) {
		req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if authToken != "" {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}
		return req, nil
	}

	// -------------------------------------------------------------------------
	// STEP 2: VIDEO INGESTION & WORKER PROCESSING
	// -------------------------------------------------------------------------
	t.Run("2. Video Ingestion & Worker Enrichment Queue", func(t *testing.T) {
		if authToken == "" {
			t.Skip("Skipping video ingestion because login failed")
		}

		// create raw video entry
		createVideoReq, _ := json.Marshal(map[string]string{
			"url": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		})

		req, err := authRequest("POST", server.URL+"/api/v1/videos", createVideoReq)
		if err != nil {
			t.Fatalf("Failed to create video request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Create video failed: %v", err)
		}
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected video creation status 201/200, got %d", resp.StatusCode)
		}

		var video VideoResponse
		json.NewDecoder(resp.Body).Decode(&video)
		resp.Body.Close()
		createdVideoID = video.ID.String()

		// enqueue enrichment job
		enrichURL := fmt.Sprintf("%s/api/v1/videos/%s/enrich", server.URL, createdVideoID)
		req, err = authRequest("POST", enrichURL, nil)
		if err != nil {
			t.Fatalf("Failed to create enrich request: %v", err)
		}

		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Enrichment enqueue failed: %v", err)
		}
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("Expected 202 Accepted on enrichment enqueue, got %d", resp.StatusCode)
		}
		resp.Body.Close()

		// poll DB/API until background worker finishes processing state
		t.Logf("Waiting for worker pool to process video %s...", createdVideoID)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

	PollLoop:
		for {
			select {
			case <-ctx.Done():
				t.Fatalf("Timed out waiting for worker pool to enrich video")
			case <-time.After(500 * time.Millisecond):
				getURL := fmt.Sprintf("%s/api/v1/videos/%s", server.URL, createdVideoID)
				resp, err := client.Get(getURL)
				if err != nil {
					continue
				}

				var pollVid VideoResponse
				json.NewDecoder(resp.Body).Decode(&pollVid)
				resp.Body.Close()

				if pollVid.Status == "analyzed" {
					break PollLoop
				}
			}
		}

		t.Logf("✓ Ingestion & Async Worker verified: Video transitioned to 'analyzed'")
	})

	// =========================================================================
	// 3. Case Milestone Linking & Timeline Fetching
	// =========================================================================
	t.Run("3. Case Milestone Linking & Timeline Fetching", func(t *testing.T) {
		// create Case
		caseReqBody := map[string]string{
			"title":       "Test Case Timeline",
			"description": "Integration test for milestone and timeline pipeline",
		}
		caseJsonBytes, _ := json.Marshal(caseReqBody)

		req, _ := authRequest("POST", fmt.Sprintf("%s/api/v1/cases", server.URL), caseJsonBytes)
		caseResp, err := client.Do(req)
		if err != nil || caseResp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to create case, status: %d", caseResp.StatusCode)
		}

		var createdCase struct {
			ID string `json:"id"`
		}
		json.NewDecoder(caseResp.Body).Decode(&createdCase)
		caseResp.Body.Close()

		// create milestone under case
		milestoneReqBody := map[string]interface{}{
			"case_id":     createdCase.ID,
			"title":       "Key Discovery Event",
			"description": "First major lead identified",
			"event_date":  time.Now().UTC().Format(time.RFC3339),
		}
		milestoneJsonBytes, _ := json.Marshal(milestoneReqBody)

		req, _ = authRequest("POST", fmt.Sprintf("%s/api/v1/cases/%s/milestones", server.URL, createdCase.ID), milestoneJsonBytes)
		milestoneResp, err := client.Do(req)
		if err != nil || milestoneResp.StatusCode != http.StatusCreated {
			t.Fatalf("Failed to create milestone, status: %d", milestoneResp.StatusCode)
		}

		var createdMilestone struct {
			ID string `json:"id"`
		}
		json.NewDecoder(milestoneResp.Body).Decode(&createdMilestone)
		milestoneResp.Body.Close()

		// link video to milestone
		linkURL := fmt.Sprintf("%s/api/v1/milestones/%s/videos/%s", server.URL, createdMilestone.ID, createdVideoID)
		req, _ = authRequest("PUT", linkURL, nil)
		linkResp, err := client.Do(req)
		if err != nil || (linkResp.StatusCode != http.StatusOK && linkResp.StatusCode != http.StatusNoContent) {
			t.Fatalf("Failed to link video to milestone, got status: %d", linkResp.StatusCode)
		}
		linkResp.Body.Close()

		// fetch case timeline & verify composite object structure
		timelineURL := fmt.Sprintf("%s/api/v1/cases/%s/timeline", server.URL, createdCase.ID)
		req, _ = authRequest("GET", timelineURL, nil)
		timelineResp, err := client.Do(req)
		if err != nil || timelineResp.StatusCode != http.StatusOK {
			t.Fatalf("Failed to fetch case timeline, got status: %d", timelineResp.StatusCode)
		}
		defer timelineResp.Body.Close()

		// match c
		// CaseTimelineResponse JSON payload format from handler
		var timelineResult struct {
			Case struct {
				ID string `json:"id"`
			} `json:"case"`
			Milestones []struct {
				ID     string `json:"id"`
				Title  string `json:"title"`
				Videos []struct {
					ID string `json:"id"`
				} `json:"videos"`
			} `json:"milestones"`
		}

		if err := json.NewDecoder(timelineResp.Body).Decode(&timelineResult); err != nil {
			t.Fatalf("Failed to decode timeline response: %v", err)
		}

		if len(timelineResult.Milestones) == 0 {
			t.Fatalf("Expected at least 1 milestone in timeline, got 0")
		}

		if len(timelineResult.Milestones[0].Videos) == 0 {
			t.Fatalf("Expected at least 1 video attached to milestone, got 0")
		}

		t.Logf("✓ Case Timeline verified: Found %d milestone(s) with %d linked video(s)",
			len(timelineResult.Milestones),
			len(timelineResult.Milestones[0].Videos),
		)
	})
}
