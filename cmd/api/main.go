package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nctqt/tc_timecapsule/internal/worker"
)

func main() {
	// root context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// initial config
	apiCfg := initialConfig()
	defer apiCfg.logFile.Close()
	defer apiCfg.db.Close()

	// bg worker pool
	workerCount := 3
	queueCapacity := 100
	apiCfg.wp = worker.NewWorkerPool(queueCapacity, apiCfg.processVideoEnrichment)

	// api
	registerEndpoints(apiCfg)

	// start worker pool
	apiCfg.wp.Start(ctx, workerCount)

	// http setup
	addr := fmt.Sprintf("%s:%s", apiCfg.httpHost, apiCfg.httpPort)
	server := &http.Server{
		Addr:    addr,
		Handler: CORSMiddleware(apiCfg.mux),
	}

	// non-blocking http server
	go func() {
		log.Printf("Server starting on :%s...", apiCfg.httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// sigint/sigterm
	<-ctx.Done()
	log.Printf("\nShutting down gracefully...")

	// stop pool and drain jobs
	apiCfg.wp.Stop()
}
