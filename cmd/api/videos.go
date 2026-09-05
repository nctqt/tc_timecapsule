package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nctqt/tc_timecapsule/internal/database"
	"github.com/nctqt/tc_timecapsule/internal/jsonhelp"
	"github.com/nctqt/tc_timecapsule/internal/openrouter"
	"github.com/nctqt/tc_timecapsule/internal/worker"
)

type CreateVideoRequest struct {
	URL         string         `json:"url"`
	MilestoneID *uuid.NullUUID `json:"milestone_id,omitempty"`
}

func (cfg *apiConfig) handlerCreateVideo(w http.ResponseWriter, r *http.Request) {
	var req CreateVideoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		// Return the exact Go decoding error to see what failed
		jsonhelp.RespondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "URL is required", errors.New("missing url"))
		return
	}

	// map pointer to uuid.NullUUID for database insertion
	var milestoneID uuid.NullUUID
	if req.MilestoneID != nil {
		milestoneID = *req.MilestoneID
	}

	// we have a url, get video meta data
	ytMeta, err := cfg.yt.FetchVideoMetaData(r.Context(), req.URL)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Failed to fetch yt metadata", err)
		return
	}

	// push to db
	now := time.Now().UTC()
	newVideo, err := cfg.queries.CreateVideo(r.Context(), database.CreateVideoParams{
		ID:             uuid.New(),
		MilestoneID:    milestoneID,
		YoutubeVideoID: ytMeta.ID,
		Title:          ytMeta.Title,
		ChannelName:    ytMeta.ChannelName,
		Description:    ytMeta.Description,
		PublishedAt:    ytMeta.PublishedAt,
		CreatedAt:      now,
		UpdatedAt:      now,
		Category:       "uncategorized",
		Status:         "pending_review", // default status
	})
	if err != nil {
		// check if the error is a pgx unique constraint violation (SQLSTATE 23505)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			jsonhelp.RespondWithError(w, http.StatusConflict, "A video with this YouTube ID already exists", err)
			return
		}

		log.Printf("Error creating video record: %v", err)
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not create video record", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusCreated, newVideo)
}

func (cfg *apiConfig) handlerListUnlinkedVideos(w http.ResponseWriter, r *http.Request) {
	videos, err := cfg.queries.ListUnlinkedVideos(r.Context())
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not gather unlinked videos", err)
		return
	}
	jsonhelp.RespondWithJSON(w, http.StatusOK, videos)
}

func (cfg *apiConfig) handlerListVideosByMilestone(w http.ResponseWriter, r *http.Request) {
	milestoneID := r.PathValue("milestone_id")
	milestoneUUID, err := uuid.Parse(milestoneID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid milestone id format", err)
		return
	}

	nullMilestoneID := uuid.NullUUID{
		UUID:  milestoneUUID,
		Valid: true,
	}

	videos, err := cfg.queries.ListVideosByMilestone(r.Context(), nullMilestoneID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not gather videos by milestone", err)
		return
	}
	jsonhelp.RespondWithJSON(w, http.StatusOK, videos)
}

func (cfg *apiConfig) handlerGetVideoByID(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("video_id")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid video id format", err)
		return
	}

	video, err := cfg.queries.GetVideoByID(r.Context(), videoUUID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not get video by id", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusOK, video)
}

func (cfg *apiConfig) handlerLinkVideoToMilestone(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("video_id")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid video id format", err)
		return
	}

	milestoneID := r.PathValue("milestone_id")
	milestoneUUID, err := uuid.Parse(milestoneID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid id format", err)
		return
	}

	nullMilestoneID := uuid.NullUUID{
		UUID:  milestoneUUID,
		Valid: true,
	}

	err = cfg.queries.LinkVideoToMilestone(r.Context(), database.LinkVideoToMilestoneParams{
		MilestoneID: nullMilestoneID,
		ID:          videoUUID,
		UpdatedAt:   time.Now(),
	})
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not link video", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerUnlinkVideoFromMilestone(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("video_id")
	videoUUID, err := uuid.Parse(videoID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid video id format", err)
		return
	}

	err = cfg.queries.UnlinkVideoFromMilestone(r.Context(), database.UnlinkVideoFromMilestoneParams{
		ID:        videoUUID,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not unlink video", err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) processVideoEnrichment(ctx context.Context, videoID uuid.UUID) (err error) {
	// if there is an error during enrichment, mark video as failed
	defer func() {
		if err != nil {
			log.Printf("[Worker] Marking video %s as failed due to error: %v", videoID, err)
			failErr := cfg.queries.UpdateVideoStatus(ctx, database.UpdateVideoStatusParams{
				ID:        videoID,
				Status:    "failed",
				UpdatedAt: time.Now().UTC(),
			})
			if failErr != nil {
				log.Printf("[Worker] Failed to set status='failed' for video %s: %v", videoID, failErr)
			}
		}
	}()
	
	// fetch raw video metadata from db
	video, err := cfg.queries.GetVideoByID(ctx, videoID)
	if err != nil {
		return fmt.Errorf("get video by id: %w", err)
	}

	// extract transcript if stored
	transcriptText := ""
	if video.RawTranscript != nil {
		transcriptText = *video.RawTranscript
	}

	// prepare request for openrouter
	analysisReq := openrouter.AnalysisRequest{
		Title:         video.Title,
		Description:   video.Description,
		ChannelName:   video.ChannelName,
		RawTranscript: transcriptText,
	}

	// call openrouter client for ai summary and category extraction
	aiResult, err := cfg.openrouter.AnalyzeVideo(ctx, analysisReq)
	if err != nil {
		return fmt.Errorf("openrouter analyze failed: %w", err)
	}

	// parse estimated time
	var estimatedEventDate sql.NullTime
	if aiResult.EstimatedEventDate != "" {
		parsedDate, err := time.Parse("2006-01-02", aiResult.EstimatedEventDate)
		if err == nil {
			estimatedEventDate = sql.NullTime{Time: parsedDate, Valid: true}
		}
	}

	now := time.Now().UTC()
	category := aiResult.Category
	if category == "" {
		category = "uncategorized"
	}

	// send to db
	_, err = cfg.queries.UpdateVideoAnalysis(ctx, database.UpdateVideoAnalysisParams{
		ID:                 video.ID,
		Category:           category,
		AiSummary:          &aiResult.Summary,
		EstimatedEventDate: estimatedEventDate,
		SummarySource:      aiResult.SummarySource,
		Status:             "analyzed", // transitions state: pending_review -> analyzed
		UpdatedAt:          now,
	})
	if err != nil {
		return fmt.Errorf("failed to update video analysis: %w", err)
	}

	return nil
}

func (cfg *apiConfig) handlerEnrichVideo(w http.ResponseWriter, r *http.Request) {
	videoIDStr := r.PathValue("video_id")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid video ID format", err)
		return
	}

	// optional quick check: ensure video exists before queuing
	_, err = cfg.queries.GetVideoByID(r.Context(), videoID)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonhelp.RespondWithError(w, http.StatusNotFound, "Video not found", err)
			return
		}
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Database error retrieving video", err)
		return
	}

	// enqueue the job for the worker pool
	enqueued := cfg.wp.Enqueue(worker.Task{VideoID: videoID})
	if !enqueued {
		jsonhelp.RespondWithError(w, http.StatusServiceUnavailable, "Enrichment queue is full", nil)
		return
	}

	// immediate 202 response
	jsonhelp.RespondWithJSON(w, http.StatusAccepted, map[string]string{
		"message":  "Video enrichment enqueued successfully",
		"video_id": videoID.String(),
		"status":   "pending_review",
	})
}
