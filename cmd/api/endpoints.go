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

	// main case data set
	apiCfg.mux.HandleFunc("GET /api/v1/cases/{case_id}/timeline", apiCfg.middlewareOptionalAuth(apiCfg.handlerGetCaseTimeline))

	// milestones
	apiCfg.mux.HandleFunc("GET /api/v1/milestones/{milestone_id}", apiCfg.handlerGetMilestoneByID)

	// videos
	apiCfg.mux.HandleFunc("GET /api/v1/videos/{video_id}", apiCfg.handlerGetVideoByID)
	apiCfg.mux.HandleFunc("GET /api/v1/milestones/{milestone_id}/videos", apiCfg.handlerListVideosByMilestone)

	// protected
	// cases
	apiCfg.mux.HandleFunc("POST /api/v1/cases", apiCfg.adminMiddleware(apiCfg.handlerCreateCase))
	apiCfg.mux.HandleFunc("DELETE /api/v1/cases/{case_id}", apiCfg.adminMiddleware(apiCfg.handlerDeleteCase))

	// timeline milestones (nested)
	apiCfg.mux.HandleFunc("POST /api/v1/cases/{case_id}/milestones", apiCfg.adminMiddleware(apiCfg.handlerCreateMilestone))

	// milestones (direct)
	apiCfg.mux.HandleFunc("DELETE /api/v1/milestones/{milestone_id}", apiCfg.adminMiddleware(apiCfg.handlerDeleteMilestone))

	// video indexing
	apiCfg.mux.HandleFunc("POST /api/v1/videos", apiCfg.adminMiddleware(apiCfg.handlerCreateVideo))
	apiCfg.mux.HandleFunc("POST /api/v1/videos/{video_id}/enrich", apiCfg.adminMiddleware(apiCfg.handlerEnrichVideo))
	apiCfg.mux.HandleFunc("GET /api/v1/videos/unlinked", apiCfg.adminMiddleware(apiCfg.handlerListUnlinkedVideos))
	apiCfg.mux.HandleFunc("PUT /api/v1/videos/{video_id}/category", apiCfg.adminMiddleware(apiCfg.handlerUpdateVideoCategory))
	apiCfg.mux.HandleFunc("PUT /api/v1/videos/{video_id}/status", apiCfg.adminMiddleware(apiCfg.handlerUpdateVideoStatus))

	// milestone linking
	apiCfg.mux.HandleFunc("PUT /api/v1/milestones/{milestone_id}/videos/{video_id}", apiCfg.adminMiddleware(apiCfg.handlerLinkVideoToMilestone))
	apiCfg.mux.HandleFunc("DELETE /api/v1/milestones/{milestone_id}/videos/{video_id}", apiCfg.adminMiddleware(apiCfg.handlerUnlinkVideoFromMilestone))

	// watch history
	apiCfg.mux.HandleFunc("POST /api/v1/watch", apiCfg.middlewareAuth(apiCfg.handlerRecordVideoWatch))
	apiCfg.mux.HandleFunc("GET /api/v1/watch/history", apiCfg.middlewareAuth(apiCfg.handlerGetWatchHistory))
}
