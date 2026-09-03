package main

import (
	"fmt"
	"log"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// initial config
	apiCfg := initialConfig()
	defer apiCfg.logFile.Close()
	defer apiCfg.db.Close()

	// api
	registerEndpoints(apiCfg)

	// http
	addr := fmt.Sprintf("%s:%s", apiCfg.httpHost, apiCfg.httpPort)
	server := &http.Server{
		Addr:    addr,
		Handler: apiCfg.mux,
	}
	log.Printf("Server running on %s", addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
