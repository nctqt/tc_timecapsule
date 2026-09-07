package main

import (
	"net/http"
)

func registerEndpoints(apiCfg *apiConfig) {
	// http router
	apiCfg.mux = http.NewServeMux()

	// public
	// system
	apiCfg.mux.HandleFunc("GET /", handlerRoot)
	apiCfg.mux.HandleFunc("GET /api/healthz", handlerHealthz)

	// authentication
	apiCfg.mux.HandleFunc("POST /api/v1/users/register", apiCfg.handlerRegisterUser)
	apiCfg.mux.HandleFunc("POST /api/v1/users/login", apiCfg.handlerLoginUser)

	// cases
	apiCfg.mux.HandleFunc("GET /api/v1/cases", apiCfg.handlerListCases)
	apiCfg.mux.HandleFunc("GET /api/v1/cases/{case_id}", apiCfg.handlerGetCaseByID)
	apiCfg.mux.HandleFunc("GET /api/v1/cases/{case_id}/milestones", apiCfg.handlerListMilestonesByCase)
	apiCfg.mux.HandleFunc("GET /api/v1/cases/{case_id}/timeline", apiCfg.handlerGetCaseTimeline)

	// milestones
	apiCfg.mux.HandleFunc("GET /api/v1/milestones/{milestone_id}", apiCfg.handlerGetMilestoneByID)

	// videos
	apiCfg.mux.HandleFunc("GET /api/v1/videos/{video_id}", apiCfg.handlerGetVideoByID)
	apiCfg.mux.HandleFunc("GET /api/v1/milestones/{milestone_id}/videos", apiCfg.handlerListVideosByMilestone)

	// protected
	// cases
	apiCfg.mux.HandleFunc("POST /api/v1/cases", apiCfg.middlewareAuth(apiCfg.handlerCreateCase))
	apiCfg.mux.HandleFunc("DELETE /api/v1/cases/{case_id}", apiCfg.middlewareAuth(apiCfg.handlerDeleteCase))

	// timeline milestones (nested)
	apiCfg.mux.HandleFunc("POST /api/v1/cases/{case_id}/milestones", apiCfg.middlewareAuth(apiCfg.handlerCreateMilestone))

	// milestones (direct)
	apiCfg.mux.HandleFunc("DELETE /api/v1/milestones/{milestone_id}", apiCfg.middlewareAuth(apiCfg.handlerDeleteMilestone))

	// video indexing
	apiCfg.mux.HandleFunc("POST /api/v1/videos", apiCfg.middlewareAuth(apiCfg.handlerCreateVideo))
	apiCfg.mux.HandleFunc("POST /api/v1/videos/{video_id}/enrich", apiCfg.middlewareAuth(apiCfg.handlerEnrichVideo))
	apiCfg.mux.HandleFunc("GET /api/v1/videos/unlinked", apiCfg.middlewareAuth(apiCfg.handlerListUnlinkedVideos))

	// milestone linking
	apiCfg.mux.HandleFunc("PUT /api/v1/milestones/{milestone_id}/videos/{video_id}", apiCfg.middlewareAuth(apiCfg.handlerLinkVideoToMilestone))
	apiCfg.mux.HandleFunc("DELETE /api/v1/milestones/{milestone_id}/videos/{video_id}", apiCfg.middlewareAuth(apiCfg.handlerUnlinkVideoFromMilestone))

	// watch history
	apiCfg.mux.HandleFunc("POST /api/v1/watch", apiCfg.middlewareAuth(apiCfg.handlerRecordVideoWatch))
	apiCfg.mux.HandleFunc("GET /api/v1/watch/history", apiCfg.middlewareAuth(apiCfg.handlerGetWatchHistory))
}
