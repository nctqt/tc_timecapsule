package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nctqt/tc_timecapsule/internal/database"
)

type CreateVideoRequest struct {
	MilestoneID    uuid.NullUUID `json:"milestone_id"`
	YoutubeVideoID string        `json:"youtube_video_id"`
	Title          string        `json:"title"`
	ChannelName    string        `json:"channel_name"`
	PublishedAt    time.Time     `json:"published_at"`
	Category       string        `json:"category"`
}

func (cfg *apiConfig) handlerCreateVideo(w http.ResponseWriter, r *http.Request) {
	var req CreateVideoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if !req.MilestoneID.Valid || req.MilestoneID.UUID == uuid.Nil {
		respondWithError(w, http.StatusBadRequest, "Invalid milestone id", err)
		return
	}

	if req.YoutubeVideoID == "" || len(req.YoutubeVideoID) > 12 {
		respondWithError(w, http.StatusBadRequest, "Invalid video id", err)
		return
	}
	req.YoutubeVideoID = strings.TrimSpace(req.YoutubeVideoID)

	if req.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Invalid video title", err)
		return
	}
	req.Title = strings.TrimSpace(req.Title)

	if req.ChannelName == "" {
		respondWithError(w, http.StatusBadRequest, "Invalid video channel name", err)
		return
	}
	req.ChannelName = strings.TrimSpace(req.ChannelName)

	if req.PublishedAt.IsZero() {
		respondWithError(w, http.StatusBadRequest, "Invalid publish time", err)
		return
	}

	switch req.Category {
	case "news", "commentary", "podcast", "analysis", "documentary", "stream", "movie":
		break
	default:
		respondWithError(w, http.StatusBadRequest, "Invalid category", err)
		return
	}

	now := time.Now().UTC()
	newVideo, err := cfg.queries.CreateVideo(r.Context(), database.CreateVideoParams{
		ID:             uuid.New(),
		MilestoneID:    req.MilestoneID,
		YoutubeVideoID: req.YoutubeVideoID,
		Title:          req.Title,
		ChannelName:    req.ChannelName,
		PublishedAt:    req.PublishedAt,
		CreatedAt:      now,
		UpdatedAt:      now,
		Category:       req.Category,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, newVideo)
}

func (cfg *apiConfig) handlerListUnlinkedVideos(w http.ResponseWriter, r *http.Request) {
	videos, err := cfg.queries.ListUnlinkedVideos(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not gather unlinked videos", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videos)
}

func (cfg *apiConfig) handlerListVideosByMilestone(w http.ResponseWriter, r *http.Request) {
	milestoneID := r.PathValue("milestone_id")
	milestoneUUID, err := uuid.Parse(milestoneID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid milestone id format", err)
		return
	}

	nullMilestoneID := uuid.NullUUID{
		UUID:  milestoneUUID,
		Valid: true,
	}

	videos, err := cfg.queries.ListVideosByMilestone(r.Context(), nullMilestoneID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not gather videos by milestone", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videos)
}
