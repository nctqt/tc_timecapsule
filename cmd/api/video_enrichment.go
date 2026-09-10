package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nctqt/tc_timecapsule/internal/database"
	"github.com/nctqt/tc_timecapsule/internal/jsonhelp"
	"github.com/nctqt/tc_timecapsule/internal/openrouter"
	"github.com/nctqt/tc_timecapsule/internal/worker"
)

func (cfg *apiConfig) processVideoEnrichment(ctx context.Context, videoID uuid.UUID) (err error) {
	// create an independent context that outlives short test/request timeouts
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	err = cfg.queries.UpdateVideoStatus(ctx, database.UpdateVideoStatusParams{
		ID:        videoID,
		Status:    "analyzing",
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		log.Printf("[Worker] Failed to set status='analyzing' for video %s: %v", videoID, err)
	}

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

	// fetch raw video metadata from DB
	video, err := cfg.queries.GetVideoByID(ctx, videoID)
	if err != nil {
		return fmt.Errorf("get video by id: %w", err)
	}

	// resolve transcript (use DB transcript if present, else fetch live via transcript client)
	transcriptText := ""
	if video.RawTranscript != nil && *video.RawTranscript != "" {
		transcriptText = *video.RawTranscript
	} else if video.YoutubeVideoID != "" {
		fetchedText, fetchErr := cfg.transcript.FetchTranscript(ctx, video.YoutubeVideoID)
		if fetchErr != nil {
			log.Printf("[Worker] Could not fetch transcript for video %s (%s): %v. Falling back to metadata.", videoID, video.YoutubeVideoID, fetchErr)
		} else {
			transcriptText = fetchedText

			// persist raw transcript to DB for future runs
			updateErr := cfg.queries.UpdateVideoTranscript(ctx, database.UpdateVideoTranscriptParams{
				ID:            video.ID,
				RawTranscript: &transcriptText,
				UpdatedAt:     time.Now().UTC(),
			})
			if updateErr != nil {
				log.Printf("[Worker] Warning: failed to save fetched transcript to DB for video %s: %v", videoID, updateErr)
			}
		}
	}

	// cap transcript length to avoid blowing through token limits on long videos
	const maxTranscriptChars = 15000
	trimmedTranscript := transcriptText
	if len(trimmedTranscript) > maxTranscriptChars {
		log.Printf("[Worker] Truncating transcript for video %s from %d to %d chars", videoID, len(trimmedTranscript), maxTranscriptChars)
		trimmedTranscript = trimmedTranscript[:maxTranscriptChars] + "\n\n[Transcript truncated due to length...]"
	}

	// prepare OpenRouter request
	analysisReq := openrouter.AnalysisRequest{
		Title:         video.Title,
		Description:   video.Description,
		ChannelName:   video.ChannelName,
		RawTranscript: trimmedTranscript,
	}

	// call OpenRouter client
	aiResult, err := cfg.openrouter.AnalyzeVideo(ctx, analysisReq)
	if err != nil {
		return fmt.Errorf("openrouter analyze failed: %w", err)
	}

	// parse estimated event date
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

	// update video analysis in database
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

	log.Printf("[Worker] Successfully enriched video %s using source: %s", video.ID, aiResult.SummarySource)
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
		"status":   "pending review",
	})
}
