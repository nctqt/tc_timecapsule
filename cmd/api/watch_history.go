package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nctqt/tc_timecapsule/internal/database"
	"github.com/nctqt/tc_timecapsule/internal/jsonhelp"
)

type RecordVideoWatchResponse struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	VideoID     uuid.UUID `json:"video_id"`
	ProgressSec int       `json:"progress_seconds"`
	IsCompleted bool      `json:"is_completed"`
	WatchedAt   time.Time `json:"watched_at"`
}

type RecordVideoWatchRequest struct {
	VideoID   uuid.UUID `json:"video_id"`
	Completed bool      `json:"completed"`
}

type contextKey string

const userIDContextKey contextKey = "userID"

func (cfg *apiConfig) handlerRecordVideoWatch(w http.ResponseWriter, r *http.Request) {
	// extract authenticated userID from context
	userID, ok := r.Context().Value(userIDContextKey).(uuid.UUID)
	if !ok {
		jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// decode request body
	var req RecordVideoWatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// validate input
	if req.VideoID == uuid.Nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "video_id is required", nil)
		return
	}

	// record watch state via sqlc query
	watchRecord, err := cfg.queries.RecordVideoWatch(r.Context(), database.RecordVideoWatchParams{
		ID:        uuid.New(),
		UserID:    userID,
		VideoID:   req.VideoID,
		WatchedAt: time.Now().UTC(),
		Completed: req.Completed,
	})
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not record watch state", err)
		return
	}

	// return created/updated record
	jsonhelp.RespondWithJSON(w, http.StatusOK, watchRecord)
}

func (cfg *apiConfig) handlerGetWatchHistory(w http.ResponseWriter, r *http.Request) {
	// extract authenticated userID from context
	userID, ok := r.Context().Value(userIDContextKey).(uuid.UUID)
	if !ok {
		jsonhelp.RespondWithError(w, http.StatusUnauthorized, "Unauthorized access", nil)
		return
	}

	// fetch watch history join records from DB
	history, err := cfg.queries.GetWatchHistoryByUserID(r.Context(), userID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not fetch watch history", err)
		return
	}

	// ensure we return an empty JSON array [] rather than null if empty
	if history == nil {
		history = []database.GetWatchHistoryByUserIDRow{}
	}

	// respond with the history items
	jsonhelp.RespondWithJSON(w, http.StatusOK, history)
}
