package main

import (
	"encoding/json"
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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
