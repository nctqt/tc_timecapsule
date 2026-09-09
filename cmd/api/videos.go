package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/nctqt/tc_timecapsule/internal/database"
	"github.com/nctqt/tc_timecapsule/internal/jsonhelp"
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

	// Pass a slice of uuid.UUID containing just this one ID
	videos, err := cfg.queries.ListVideosByMilestoneIDs(r.Context(), []uuid.UUID{milestoneUUID})
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
		if errors.Is(err, sql.ErrNoRows) {
			jsonhelp.RespondWithError(w, http.StatusNotFound, "Video not found", err)
			return
		}
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
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not link video", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		UpdatedAt: time.Now().UTC(),
	})
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not unlink video", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
