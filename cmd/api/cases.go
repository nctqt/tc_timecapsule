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

type createCaseRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"` // optional
}

func (cfg *apiConfig) handlerCreateCase(w http.ResponseWriter, r *http.Request) {
	var req createCaseRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if req.Title == "" {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Title is required", nil)
		return
	}

	now := time.Now().UTC()
	newCase, err := cfg.queries.CreateCase(r.Context(), database.CreateCaseParams{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description, // directly maps to sql.NullString / *string depending on sqlc settings
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not create case", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusCreated, newCase)
}

func (cfg *apiConfig) handlerListCases(w http.ResponseWriter, r *http.Request) {
	cases, err := cfg.queries.ListCases(r.Context())
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not list cases", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusOK, cases)
}

func (cfg *apiConfig) handlerGetCaseByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("case_id")
	caseUUID, err := uuid.Parse(id)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid id format", err)
		return
	}

	caseRecord, err := cfg.queries.GetCaseByID(r.Context(), caseUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			jsonhelp.RespondWithError(w, http.StatusNotFound, "Case not found", err)
			return
		}
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not retrieve case", err)
		return
	}

	jsonhelp.RespondWithJSON(w, http.StatusOK, caseRecord)
}

func (cfg *apiConfig) handlerDeleteCase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("case_id")
	caseUUID, err := uuid.Parse(id)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid id format", err)
		return
	}

	err = cfg.queries.DeleteCase(r.Context(), caseUUID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not delete case", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type MilestoneWithVideos struct {
	database.Milestone
	Videos []database.Video `json:"videos"`
}

type CaseTimelineResponse struct {
	Case       database.Case         `json:"case"`
	Milestones []MilestoneWithVideos `json:"milestones"`
}

func (cfg *apiConfig) handlerGetCaseTimeline(w http.ResponseWriter, r *http.Request) {
	caseIDStr := r.PathValue("case_id")
	caseID, err := uuid.Parse(caseIDStr)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusBadRequest, "Invalid case id format", err)
		return
	}

	// 1. Fetch case details
	caseData, err := cfg.queries.GetCaseByID(r.Context(), caseID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusNotFound, "Case not found", err)
		return
	}

	// 2. Fetch milestones for the case
	milestones, err := cfg.queries.ListMilestonesByCase(r.Context(), caseID)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not fetch milestones", err)
		return
	}

	if len(milestones) == 0 {
		jsonhelp.RespondWithJSON(w, http.StatusOK, CaseTimelineResponse{
			Case:       caseData,
			Milestones: []MilestoneWithVideos{},
		})
		return
	}

	// 3. Collect ALL milestone UUIDs into a slice
	milestoneIDs := make([]uuid.UUID, len(milestones))
	for i, m := range milestones {
		milestoneIDs[i] = m.ID
	}

	// 4. Batch fetch videos for ALL milestones at once
	videos, err := cfg.queries.ListVideosByMilestoneIDs(r.Context(), milestoneIDs)
	if err != nil {
		jsonhelp.RespondWithError(w, http.StatusInternalServerError, "Could not fetch videos", err)
		return
	}

	// 5. Group videos by milestone ID
	videosByMilestone := make(map[uuid.UUID][]database.Video)
	for _, v := range videos {
		if v.MilestoneID.Valid {
			videosByMilestone[v.MilestoneID.UUID] = append(videosByMilestone[v.MilestoneID.UUID], v)
		}
	}

	// 6. Build the nested response
	milestonesWithVideos := make([]MilestoneWithVideos, len(milestones))
	for i, m := range milestones {
		vList := videosByMilestone[m.ID]
		if vList == nil {
			vList = []database.Video{}
		}
		milestonesWithVideos[i] = MilestoneWithVideos{
			Milestone: m,
			Videos:    vList,
		}
	}

	jsonhelp.RespondWithJSON(w, http.StatusOK, CaseTimelineResponse{
		Case:       caseData,
		Milestones: milestonesWithVideos,
	})
}
