package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nctqt/tc_timecapsule/internal/database"
	"github.com/nctqt/tc_timecapsule/internal/jsonhelp"
)

type createMilestoneRequest struct {
	Title         string    `json:"title"`
	Description   *string   `json:"description"`
	DatePrecision *string   `json:"date_precision"`
	EventDate     time.Time `json:"event_date"`
}

func (cfg *apiConfig) handlerCreateMilestone(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("case_id")
	caseUUID, err := uuid.Parse(caseID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid id format", err)
		return
	}

	var req createMilestoneRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.Title == "" {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Title is required", nil)
		return
	}

	var precision string
	if req.DatePrecision != nil {
		precision = *req.DatePrecision
	}
	switch precision {
	case "exact", "year", "month", "day":
		break
	default:
		precision = "exact"
	}

	now := time.Now().UTC()
	newMilestone, err := cfg.queries.CreateMilestone(r.Context(), database.CreateMilestoneParams{
		ID:            uuid.New(),
		CaseID:        caseUUID,
		Title:         req.Title,
		Description:   req.Description,
		EventDate:     req.EventDate,
		DatePrecision: precision,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not create milestone", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusCreated, newMilestone)
}

func (cfg *apiConfig) handlerListMilestonesByCase(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("case_id")
	caseUUID, err := uuid.Parse(caseID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid id format", err)
		return
	}

	milestones, err := cfg.queries.ListMilestonesByCase(r.Context(), caseUUID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not list milestones", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusOK, milestones)
}

func (cfg *apiConfig) handlerGetMilestoneByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("milestone_id")
	UUID, err := uuid.Parse(id)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid id format", err)
		return
	}

	milestone, err := cfg.queries.GetMilestoneByID(r.Context(), UUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonhelp.RespondWithError(w, http.StatusNotFound, "Milestone not found", err)
			return
		}
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not retrieve milestone", err)
		return
	}
	jsonhelp.RespondWithJSON(w, http.StatusOK, milestone)
}

func (cfg *apiConfig) handlerDeleteMilestone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("milestone_id")
	UUID, err := uuid.Parse(id)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid id format", err)
		return
	}

	err = cfg.queries.DeleteMilestone(r.Context(), UUID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not delete milestone", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
