package main

import (
	"net/http"
)

func registerEndpoints(apiCfg *apiConfig) {
	// http router
	apiCfg.mux = http.NewServeMux()

	// root and system
	apiCfg.mux.HandleFunc("GET /", handlerRoot)
	apiCfg.mux.HandleFunc("GET /api/healthz", handlerHealthz)

	// cases endpoints
	apiCfg.mux.HandleFunc("POST /api/v1/cases", apiCfg.handlerCreateCase)
	apiCfg.mux.HandleFunc("GET /api/v1/cases", apiCfg.handlerListCases)
	apiCfg.mux.HandleFunc("GET /api/v1/cases/{case_id}", apiCfg.handlerGetCaseByID)
	apiCfg.mux.HandleFunc("DELETE /api/v1/cases/{case_id}", apiCfg.handlerDeleteCase)

	// timeline milestones endpoints (nested under cases)
	apiCfg.mux.HandleFunc("POST /api/v1/cases/{case_id}/milestones", apiCfg.handlerCreateMilestone)
	apiCfg.mux.HandleFunc("GET /api/v1/cases/{case_id}/milestones", apiCfg.handlerListMilestonesByCase)

	// milestones not nested
	apiCfg.mux.HandleFunc("GET /api/v1/milestones/{milestone_id}", apiCfg.handlerGetMilestoneByID)
	apiCfg.mux.HandleFunc("DELETE /api/v1/milestones/{milestone_id}", apiCfg.handlerDeleteMilestone)

	// video indexing endpoints
	apiCfg.mux.HandleFunc("POST /api/v1/videos", apiCfg.handlerCreateVideo)
	apiCfg.mux.HandleFunc("POST /api/v1/videos/{video_id}/enrich", apiCfg.handlerEnrichVideo)
	apiCfg.mux.HandleFunc("GET /api/v1/videos/unlinked", apiCfg.handlerListUnlinkedVideos)
	apiCfg.mux.HandleFunc("GET /api/v1/videos/{video_id}", apiCfg.handlerGetVideoByID)
	apiCfg.mux.HandleFunc("GET /api/v1/milestones/{milestone_id}/videos", apiCfg.handlerListVideosByMilestone)

	// milestone linking
	apiCfg.mux.HandleFunc("PUT /api/v1/milestones/{milestone_id}/videos/{video_id}", apiCfg.handlerLinkVideoToMilestone)
	apiCfg.mux.HandleFunc("DELETE /api/v1/milestones/{milestone_id}/videos/{video_id}", apiCfg.handlerUnlinkVideoFromMilestone)
}
